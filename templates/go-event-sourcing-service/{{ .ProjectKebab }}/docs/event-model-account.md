# Event Model: Bank Account

## Overview

The example domain of this template is a bank account. A client opens an account, puts money in, takes
money out and closes the account through the HTTP API. A reaction gives each new account a welcome
bonus. A projection keeps the balance of each account in the `account_balances` table.

- Model: [event-model-account.emod](event-model-account.emod), in the [emod](https://github.com/hpcsc/emod) DSL
- Diagram: [event-model-account.svg](event-model-account.svg), drawn from the model

When the project has the money transfer example, [event-model-transfer.md](event-model-transfer.md)
describes the full model: this account model plus the transfer.

Check the model and draw the diagram again after a change:

```sh
emod validate docs/event-model-account.emod
emod lint docs/event-model-account.emod
emod diagram docs/event-model-account.emod --format svg --specs -o docs/event-model-account.svg
```

`emod lint` reports three warnings. [Lint Warnings](#lint-warnings) tells why the model keeps them.

## Actors

| Actor | Type | Description |
| --- | --- | --- |
| Client | System or human | A program or a person that calls the HTTP API with a bearer token |
| `WelcomeBonusReaction` | Automation | Gives the welcome bonus to each new account |

## Storyboard

The HTTP API is the only touchpoint. Each route that sends a command answers before the projection
changes the balance, so `GET /accounts/{id}` can show the change a short time later.

| Route | Inputs | Output |
| --- | --- | --- |
| `POST /accounts` | `owner` in the body | The new account ID |
| `POST /accounts/{id}/deposits` | `amount` in the body | `200 OK` with no data |
| `POST /accounts/{id}/withdrawals` | `amount` in the body | `200 OK` with no data |
| `POST /accounts/{id}/close` | None | `200 OK` with no data |
| `GET /accounts/{id}` | None | The owner, the balance, the status and the version |

The client reads the account with `GET /accounts/{id}` before a deposit, a withdrawal or a close. The
model shows this as a `reads` from each of the three triggers to `AccountBalanceView`.

## Business Processes

| Capability | Process | Trigger | Steps | Outcome |
| --- | --- | --- | --- | --- |
| Account keeping | Open an account | `POST /accounts` | Open → bonus | An account with a balance of 1000 |
| Account keeping | Move money | `POST /accounts/{id}/deposits` or `/withdrawals` | Deposit or withdraw | A new balance |
| Account keeping | Close an account | `POST /accounts/{id}/close` | Close | An account that takes no more money |

### Timeline

```
POST /accounts          -> OpenAccount       -> AccountOpened
AccountOpened           -> WelcomeBonusReaction -> GrantWelcomeBonus -> WelcomeBonusGranted
POST .../deposits       -> Deposit           -> MoneyDeposited
POST .../withdrawals    -> Withdraw          -> MoneyWithdrawn
POST .../close          -> CloseAccount      -> AccountClosed
all five events         -> AccountBalanceView (GET /accounts/{id})
```

### Events

| Event | Description | Key data |
| --- | --- | --- |
| `AccountOpened` | An account was opened for an owner | `accountId`, `owner` |
| `WelcomeBonusGranted` | A new account got the welcome bonus | `accountId`, `amount` |
| `MoneyDeposited` | Money went into an account | `accountId`, `amount` |
| `MoneyWithdrawn` | Money went out of an account | `accountId`, `amount` |
| `AccountClosed` | An account was closed | `accountId` |

The store records the time of each event in the `created_at` column, so no payload carries a time.

### Commands

| Command | Actor | Produces | Refused when | Required data |
| --- | --- | --- | --- | --- |
| `OpenAccount` | Client | `AccountOpened` | `OpenedOnce`, `HasOwner` | `accountId`: made by the API. `owner`: the body |
| `GrantWelcomeBonus` | `WelcomeBonusReaction` | `WelcomeBonusGranted` | `AccountExists`, `PositiveAmount` | `accountId`: from `AccountOpened`. `amount`: `welcomebonus.Amount`, 1000 |
| `Deposit` | Client | `MoneyDeposited` | `AccountExists`, `NotClosed`, `PositiveAmount` | `accountId`: the path. `amount`: the body |
| `Withdraw` | Client | `MoneyWithdrawn` | `AccountExists`, `NotClosed`, `PositiveAmount`, `EnoughFunds` | `accountId`: the path. `amount`: the body |
| `CloseAccount` | Client | `AccountClosed` | `AccountExists`, `ClosedOnce` | `accountId`: the path |

### Rules

| Rule | Statement | Code |
| --- | --- | --- |
| `OpenedOnce` | An account is opened once | `account.ErrAlreadyOpened` |
| `HasOwner` | An account has an owner | The error in `Account.Open` |
| `AccountExists` | A command changes only an account that was opened | `account.ErrNotOpened` |
| `NotClosed` | A closed account takes no deposit and no withdrawal | `account.ErrClosed` |
| `PositiveAmount` | An amount is more than 0 | `account.ErrInvalidAmount` |
| `EnoughFunds` | A withdrawal is at most the balance | `account.ErrInsufficientFunds` |
| `ClosedOnce` | An account is closed once | `account.ErrClosed` |

A rule refuses the command, and the API answers `400 Bad Request`.

`GrantWelcomeBonus` records nothing in two cases, and the specs show them with `then = []`:

- The account has the bonus.
- The account is closed.

### Views

| View | Purpose | Backs | Data lineage |
| --- | --- | --- | --- |
| `AccountBalanceView` | One account with its owner, balance and status | `GET /accounts/{id}` | `id`, `owner` ← `AccountOpened`. `balance` ← `+ WelcomeBonusGranted.amount`, `+ MoneyDeposited.amount`, `- MoneyWithdrawn.amount`. `status` ← `closed` after `AccountClosed`. `version` ← the stream version of the last event |

### Automations

| Trigger event | Automation | Command | Result event |
| --- | --- | --- | --- |
| `AccountOpened` | `WelcomeBonusReaction` | `GrantWelcomeBonus` | `WelcomeBonusGranted` |

### Message Processing Patterns

| # | From | To | Pattern | Component |
| --- | --- | --- | --- | --- |
| 1 | `POST /accounts` | `OpenAccount` | UI trigger | `internal/api/use_cases/account` |
| 2 | `OpenAccount` | `AccountOpened` | Aggregate | `internal/use_cases/account/open` |
| 3 | `AccountOpened` | `GrantWelcomeBonus` | Reactor | `welcomebonus.Reaction`, consumer `welcome-bonus` |
| 4 | `GrantWelcomeBonus` | `WelcomeBonusGranted` | Aggregate | `internal/use_cases/account/welcomebonus` |
| 5 | `POST .../deposits` | `Deposit` | UI trigger | `internal/api/use_cases/account` |
| 6 | `Deposit` | `MoneyDeposited` | Aggregate | `internal/use_cases/account/deposit` |
| 7 | `POST .../withdrawals` | `Withdraw` | UI trigger | `internal/api/use_cases/account` |
| 8 | `Withdraw` | `MoneyWithdrawn` | Aggregate | `internal/use_cases/account/withdraw` |
| 9 | `POST .../close` | `CloseAccount` | UI trigger | `internal/api/use_cases/account` |
| 10 | `CloseAccount` | `AccountClosed` | Aggregate | `internal/use_cases/account/closeaccount` |
| 11 | The five events | `AccountBalanceView` | Projection | `balance.Projection`, consumer `account-balances` |

The aggregate is `account.Account` in `internal/use_cases/account/account.go`. The `catchup` process runs
the reaction and the projection.

## Autonomous Components

| Component | Owned events | Owned commands | Consumes events from |
| --- | --- | --- | --- |
| Accounts | All five events | All five commands | None |

## Implementation Slices

| # | Slice | Events | Commands | Views | Package |
| --- | --- | --- | --- | --- | --- |
| 1 | Open Account | `AccountOpened` | `OpenAccount` | | `internal/use_cases/account/open` |
| 2 | Grant Welcome Bonus | `WelcomeBonusGranted` | `GrantWelcomeBonus` | | `internal/use_cases/account/welcomebonus` |
| 3 | Deposit | `MoneyDeposited` | `Deposit` | | `internal/use_cases/account/deposit` |
| 4 | Withdraw | `MoneyWithdrawn` | `Withdraw` | | `internal/use_cases/account/withdraw` |
| 5 | Close Account | `AccountClosed` | `CloseAccount` | | `internal/use_cases/account/closeaccount` |
| 6 | View Account Balance | | | `AccountBalanceView` | `internal/use_cases/account/balance` |

## Lint Warnings

| Rule | Element | Why the model keeps it |
| --- | --- | --- |
| `automation/missing-todo-list` | `WelcomeBonusReaction` | The reaction reads no view. It keeps a checkpoint, and the account aggregate records the bonus once, so no list of work is necessary. |
| `clickbait-event` | `AccountClosed` | The name tells exactly what happened, and the balance projection sets the status without a lookup. The rule checks only that the payload holds one ID. |
| `god-view` | `AccountBalanceView` | A balance must include each money movement of the account. Each event is a different business fact. One `BalanceChanged` event would be the State Obsession anti-pattern. |
