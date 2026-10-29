# Event Model: Bank Account and Money Transfer

## Overview

This model is the full example domain of a project with the money transfer example. It holds the account
model of [event-model-account.md](event-model-account.md) and adds a transfer of money between two
accounts. A process manager runs each transfer. A failed credit refunds the debit.
[process-manager-example.md](process-manager-example.md) explains the design of the process manager.

- Model: [event-model-transfer.emod](event-model-transfer.emod), in the [emod](https://github.com/hpcsc/emod) DSL
- Diagram: [event-model-transfer.svg](event-model-transfer.svg), drawn from the model

This document describes only what the transfer adds. The account part is the same as in
[event-model-account.md](event-model-account.md), with three more events in `AccountBalanceView`.

Check the model and draw the diagram again after a change:

```sh
emod validate docs/event-model-transfer.emod
emod lint docs/event-model-transfer.emod
emod diagram docs/event-model-transfer.emod --format svg --specs -o docs/event-model-transfer.svg
```

`emod lint` reports five warnings. [Lint Warnings](#lint-warnings) tells why the model keeps them.

## Actors

| Actor | Type | Description |
| --- | --- | --- |
| Client | System or human | A program or a person that calls the HTTP API with a bearer token |
| `TransferProcess` | Automation | The process manager that runs each transfer to its end |

## Storyboard

| Route | Inputs | Output |
| --- | --- | --- |
| `POST /transfers` | `from`, `to` and `amount` in the body | The new transfer ID |
| `GET /transfers/{id}` | None | The two accounts, the amount, the state, and the reason of a failure |

`GET /transfers/{id}` reads `TransferProcessView`. The state is `debiting`, `crediting`, `refunding`,
`completed` or `failed`.

## Business Process: Move Money Between Two Accounts

| Trigger | Steps | Outcome |
| --- | --- | --- |
| `POST /transfers` | Start → debit the source → credit the destination → complete. Or: debit refused → fail. Or: credit refused → refund the source → fail | The money moved, or the source account has its money back and the transfer has a reason |

### Timeline

```
POST /transfers                         -> StartTransfer     -> TransferStarted
TransferStarted        -> TransferProcess -> DebitForTransfer  -> TransferDebited or TransferDebitRejected
TransferDebited        -> TransferProcess -> CreditForTransfer -> TransferCredited or TransferCreditRejected
TransferCredited       -> TransferProcess -> CompleteTransfer  -> TransferCompleted
TransferCreditRejected -> TransferProcess -> RefundTransfer    -> TransferRefunded
TransferDebitRejected  -> TransferProcess -> FailTransfer      -> TransferFailed
TransferRefunded       -> TransferProcess -> FailTransfer      -> TransferFailed
```

### Events

| Event | Stream | Description | Key data |
| --- | --- | --- | --- |
| `TransferStarted` | `transfer-<id>` | A transfer from one account to another started | `transferId`, `from`, `to`, `amount` |
| `TransferDebited` | `account-<id>` | The source account gave the amount | `accountId`, `transferId`, `amount` |
| `TransferDebitRejected` | `account-<id>` | The source account cannot give the amount | `accountId`, `transferId`, `amount`, `reason` |
| `TransferCredited` | `account-<id>` | The destination account took the amount | `accountId`, `transferId`, `amount` |
| `TransferCreditRejected` | `account-<id>` | The destination account cannot take the amount | `accountId`, `transferId`, `amount`, `reason` |
| `TransferRefunded` | `account-<id>` | The source account got the amount back | `accountId`, `transferId`, `amount` |
| `TransferCompleted` | `transfer-<id>` | The money moved from one account to the other | `transferId` |
| `TransferFailed` | `transfer-<id>` | The money did not move | `transferId`, `reason` |

### Commands

| Command | Actor | Produces | Refused when | Required data |
| --- | --- | --- | --- | --- |
| `StartTransfer` | Client | `TransferStarted` | `StartedOnce`, `TwoAccounts`, `PositiveAmount`, `AccountsExist` | `transferId`: made by the API. `from`, `to`, `amount`: the body |
| `DebitForTransfer` | `TransferProcess` | `TransferDebited` or `TransferDebitRejected` | `AccountExists` | `accountId` ← `from`. `transferId`, `amount` ← `TransferProcessView` |
| `CreditForTransfer` | `TransferProcess` | `TransferCredited` or `TransferCreditRejected` | `AccountExists` | `accountId` ← `to`. `transferId`, `amount` ← `TransferProcessView` |
| `RefundTransfer` | `TransferProcess` | `TransferRefunded` | `AccountExists` | `accountId` ← `from`. `transferId`, `amount` ← `TransferProcessView` |
| `CompleteTransfer` | `TransferProcess` | `TransferCompleted` | `EndsOnce` | `transferId` ← `TransferProcessView` |
| `FailTransfer` | `TransferProcess` | `TransferFailed` | `EndsOnce` | `transferId` ← `TransferProcessView`. `reason` ← `TransferDebitRejected.reason`, or the reason that `TransferProcessView` keeps from `TransferCreditRejected` |

### Rules

| Rule | Aggregate | Statement | Code |
| --- | --- | --- | --- |
| `StartedOnce` | Transfer | A transfer is started once | `transfer.ErrAlreadyStarted` |
| `TwoAccounts` | Transfer | A transfer moves money between two different accounts | `transfer.ErrSameAccount` |
| `PositiveAmount` | Transfer | An amount is more than 0 | `transfer.ErrInvalidAmount` |
| `AccountsExist` | Transfer | Both accounts of a transfer were opened | `transfer.ErrUnknownAccount` |
| `EndsOnce` | Transfer | A transfer ends once: it completes or it fails | The state check in `process.Process.Advance` |
| `AccountExists` | Account | A command changes only an account that was opened | `account.ErrNotOpened` |

A failure that the process acts on is an event, not a refusal:

- `TransferDebitRejected`: the source account is closed, the amount is not more than 0, or the balance is
  less than the amount.
- `TransferCreditRejected`: the destination account is closed.

Each account decides once for each transfer. A repeated debit, credit or refund for the same transfer
records nothing, and the specs show this with `then = []`.

### Views

| View | Purpose | Backs | Data lineage |
| --- | --- | --- | --- |
| `AccountBalanceView` | One account with its owner, balance and status | `GET /accounts/{id}` | The lineage of [event-model-account.md](event-model-account.md), plus `balance` ← `- TransferDebited.amount`, `+ TransferCredited.amount`, `+ TransferRefunded.amount` |
| `TransferProcessView` | The state of each transfer: the list of work of `TransferProcess` | `GET /transfers/{id}`, `TransferProcess` | `id`, `from`, `to`, `amount` ← `TransferStarted`. `state` ← `debiting` after `TransferStarted`, `crediting` after `TransferDebited`, `refunding` after `TransferCreditRejected`, `completed` after `TransferCredited`, `failed` after `TransferDebitRejected` or `TransferRefunded`. `reason` ← `TransferDebitRejected.reason` or `TransferCreditRejected.reason`. `streamVersion` ← the version of the transfer stream |

### Automations

| Trigger event | Automation | Command | Result event |
| --- | --- | --- | --- |
| `TransferStarted` | `TransferProcess` | `DebitForTransfer` | `TransferDebited` or `TransferDebitRejected` |
| `TransferDebited` | `TransferProcess` | `CreditForTransfer` | `TransferCredited` or `TransferCreditRejected` |
| `TransferCredited` | `TransferProcess` | `CompleteTransfer` | `TransferCompleted` |
| `TransferCreditRejected` | `TransferProcess` | `RefundTransfer` | `TransferRefunded` |
| `TransferDebitRejected` | `TransferProcess` | `FailTransfer` | `TransferFailed` |
| `TransferRefunded` | `TransferProcess` | `FailTransfer` | `TransferFailed` |

### Message Processing Patterns

| # | From | To | Pattern | Component |
| --- | --- | --- | --- | --- |
| 1 | `POST /transfers` | `StartTransfer` | UI trigger | `internal/api/use_cases/transfer` |
| 2 | `StartTransfer` | `TransferStarted` | Aggregate | `internal/use_cases/transfer/start` |
| 3 | The start and the five account replies | `TransferProcessView` | Projection | `process.Projection`, consumer `transfer-process` |
| 4 | `TransferStarted`, `TransferDebited`, `TransferCreditRejected` | `DebitForTransfer`, `CreditForTransfer`, `RefundTransfer` | Process manager | `process.Projection` decides, `process.Dispatcher` sends, consumer `transfer-requests` |
| 5 | `DebitForTransfer`, `CreditForTransfer`, `RefundTransfer` | The account replies | Aggregate | `internal/use_cases/account/transfers` |
| 6 | `TransferCredited`, `TransferDebitRejected`, `TransferRefunded` | `TransferCompleted`, `TransferFailed` | Process manager | `process.Projection` |
| 7 | The three transfer events of the accounts | `AccountBalanceView` | Projection | `balance.Projection`, consumer `account-balances` |

## Where the Model and the Code Are Different

The model shows the process manager in the usual Event Modeling form: a view, and an automation that
reads it and sends a command. The code has the same steps, with these differences:

- **The request events are not in the model.** `process.Projection` appends each decision to the transfer
  stream as a request event: `TransferDebitRequested`, `TransferCreditRequested` or
  `TransferRefundRequested`. It does this in the same transaction as the state. `process.Dispatcher` then
  reads each request and sends the account command. Thus a crash between the decision and the send loses
  no step. The request events tell how the code sends a command, not what happened to the money, so the
  model leaves them out.
- **`CompleteTransfer` and `FailTransfer` have no type in the code.** `process.Projection` appends
  `TransferCompleted` and `TransferFailed` itself. emod needs a command for each event.
- **`EndsOnce` is a state check.** `process.Process.Advance` takes a step only from the state that waits
  for the reply, so a reply that arrives again asks for nothing. The model shows this as a rule of the
  transfer aggregate.
- **`AccountsExist` reads other streams.** The start handler loads the two account streams before it
  starts the transfer. A spec can show only the events of its own aggregate, so its spec has no account
  history.
- **Three `TransferProcess` automations are in the Accounts context.** emod puts each automation in the
  slice of the command that it sends. In the code, the whole process is in
  `internal/use_cases/transfer/process`.

## Autonomous Components

| Component | Owned events | Owned commands | Consumes events from |
| --- | --- | --- | --- |
| Accounts | The account events of [event-model-account.md](event-model-account.md), plus `TransferDebited`, `TransferDebitRejected`, `TransferCredited`, `TransferCreditRejected`, `TransferRefunded` | The account commands, plus `DebitForTransfer`, `CreditForTransfer`, `RefundTransfer` | None. `TransferProcess` sends the commands |
| Transfers | `TransferStarted`, `TransferCompleted`, `TransferFailed` | `StartTransfer`, `CompleteTransfer`, `FailTransfer` | The five transfer events of Accounts |

## Implementation Slices

| # | Slice | Events | Commands | Views | Package |
| --- | --- | --- | --- | --- | --- |
| 1 | Start Transfer | `TransferStarted` | `StartTransfer` | | `internal/use_cases/transfer/start` |
| 2 | Debit for Transfer | `TransferDebited`, `TransferDebitRejected` | `DebitForTransfer` | | `internal/use_cases/account/transfers` |
| 3 | Credit for Transfer | `TransferCredited`, `TransferCreditRejected` | `CreditForTransfer` | | `internal/use_cases/account/transfers` |
| 4 | Refund Transfer | `TransferRefunded` | `RefundTransfer` | | `internal/use_cases/account/transfers` |
| 5 | View Transfer Process | | | `TransferProcessView` | `internal/use_cases/transfer/process` |
| 6 | Complete Transfer | `TransferCompleted` | `CompleteTransfer` | | `internal/use_cases/transfer/process` |
| 7 | Fail Transfer | `TransferFailed` | `FailTransfer` | | `internal/use_cases/transfer/process` |

## Lint Warnings

| Rule | Element | Why the model keeps it |
| --- | --- | --- |
| `automation/missing-todo-list` | `WelcomeBonusReaction` | The same as in [event-model-account.md](event-model-account.md#lint-warnings) |
| `clickbait-event` | `AccountClosed` | The same as in [event-model-account.md](event-model-account.md#lint-warnings) |
| `god-view` | `AccountBalanceView` | A balance must include each money movement of the account. With the transfer, that is 8 events, and each is a different business fact |
| `god-view` | `TransferProcessView` | A process manager must receive the start and each of the five replies from the accounts |
| `clickbait-event` | `TransferCompleted` | The name tells exactly what happened, and no code reads it. A later consumer outside the transfer stream gets `from`, `to` and `amount` from `TransferStarted` |
