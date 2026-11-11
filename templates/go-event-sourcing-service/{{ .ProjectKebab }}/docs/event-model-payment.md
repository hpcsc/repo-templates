# Event Model: Bank Account and Payment to Another Bank

## Overview

This model is the full example domain of a project with the payment example. It holds the account model
of [event-model-account.md](event-model-account.md), the holds of the account, and a payment from an
account to an account at another bank. The bank holds the amount, checks the payment, sends it to the
payment network, and waits for the network to settle it.

The payment aggregate, with its reactions and gateways, is the process manager of the payment:

- The payment aggregate holds the state of the payment and decides each step. Each input reaches it as a
  command, and each decision leaves it as an event.
- The reactions and the gateways carry the messages between the payment, the account and the outside
  systems. They make no decision.
- The webhook routes turn the answers of the outside systems into commands for the payment.
- A scheduler reads a todo list of deadlines, and tells the payment when a deadline passed.

[process-manager-example.md](process-manager-example.md) explains the design of the process manager.

- Model: [event-model-payment.emod](event-model-payment.emod), in the [emod](https://github.com/hpcsc/emod) DSL
- Diagram: [event-model-payment.svg](event-model-payment.svg), drawn from the model

This document describes only what the payment adds. The account part is the same as in
[event-model-account.md](event-model-account.md), with these changes:

- The account has three more commands: `PlaceHold`, `ReleaseHold` and `CaptureHold`.
- `EnoughFunds` checks the available balance: the balance minus the holds.
- `AccountBalanceView` has one more event: `HoldCaptured`.

Check the model and draw the diagram again after a change:

```sh
emod validate docs/event-model-payment.emod
emod lint docs/event-model-payment.emod
emod diagram docs/event-model-payment.emod --format svg --specs -o docs/event-model-payment.svg
```

`emod lint` reports 20 warnings. [Lint Warnings](#lint-warnings) tells why the model keeps them.

## Actors

| Actor | Type | Description |
| --- | --- | --- |
| Client | System or human | A program or a person that calls the HTTP API with a bearer token |
| Fraud service | External system | Checks a payment for fraud, and sends the result to `POST /api/webhooks/fraud-checks` |
| Sanctions service | External system | Checks the creditor against the sanctions lists, and sends the result to `POST /api/webhooks/sanctions-screenings` |
| Payment network | External system | Takes the payment to the other bank. It sends the settlement or the return to `POST /api/webhooks/payment-network`, and it answers a status request |
| `PaymentFundingReaction` | Automation | Asks the account to hold the amount, and tells the payment the answer of the account |
| `PaymentHoldsReaction` | Automation | Tells the account to capture or release the hold when the payment ends |
| `PaymentChecksGateway` | Automation | Sends the payment to the fraud service and to the sanctions service |
| `PaymentSubmissionGateway` | Automation | Sends the payment to the payment network, and tells the payment the reference of the network |
| `PaymentStatusGateway` | Automation | Asks the payment network for the status of the payment, and tells the payment the answer |
| `PaymentTimeoutsScheduler` | Automation | Each second, tells the payment about each deadline that passed |

In this template, `cmd/fakeservices` fakes the three outside systems, and the input of the payment
selects their answers. The specs use the reasons that the fakes send, for example
`creditor on the sanctions list`.

## Storyboard

The HTTP API is the touchpoint of the client and of the outside systems.

| Route | Caller | Inputs | Output |
| --- | --- | --- | --- |
| `POST /api/payments` | Client | `accountId`, `creditorIban` and `amount` in the body | The new payment ID |
| `GET /api/payments/{id}` | Client | None | The account ID, the creditor IBAN, the amount, the status, the reason, the network reference and the version |
| `POST /api/webhooks/fraud-checks` | Fraud service | `paymentId`, `passed` and `reason` in the body | `200 OK` with no data |
| `POST /api/webhooks/sanctions-screenings` | Sanctions service | `paymentId`, `passed` and `reason` in the body | `200 OK` with no data |
| `POST /api/webhooks/payment-network` | Payment network | `paymentId`, `status` and `reason` in the body. The status is `settled` or `returned` | `200 OK` with no data |

- All five routes are protected routes, so an outside system sends the bearer token of the API.
- `POST /api/payments` answers before the payment ends. The client reads the progress of the payment with
  `GET /api/payments/{id}`. The model shows this read as the trigger `GET /api/payments/{id}`, with no command.
- The status is `requested`, `funded`, `cleared`, `submitted`, `escalated`, `settled`, `returned`,
  `refused` or `failed`. The payment has a reason when its status is `failed`, `refused`, `escalated` or
  `returned`.
- A refused `POST /api/payments` gets `400 Bad Request`. A webhook for a payment that does not exist gets
  `404 Not Found`.

## Business Process: Pay an Account at Another Bank

Each path starts with `POST /api/payments`.

| Path | Steps | Outcome |
| --- | --- | --- |
| Settled | Request → hold → both checks pass → clear → submit → settle → capture the hold | The amount left the account and the bank |
| Not funded | Request → the account rejects the hold → fail | The account has no hold |
| Refused | Request → hold → a check fails → refuse → release the hold | The amount is available again |
| Checks with no answer | Request → hold → a check service does not answer after the last attempt → fail → release the hold | The amount is available again |
| Returned | Request → hold → both checks pass → clear → submit → return → release the hold | The amount is available again |
| Escalated | Request → hold → both checks pass → clear → the network does not answer after the last attempt → escalate | The hold stays. A person checks the payment, and a later webhook of the network settles or returns it |

### Timeline

```
POST /api/payments                                 -> RequestPayment           -> PaymentRequested
PaymentRequested                                   -> PaymentFundingReaction   -> PlaceHold             -> HoldPlaced or HoldRejected
HoldPlaced                                         -> PaymentFundingReaction   -> RecordFunding         -> PaymentFunded
HoldRejected                                       -> PaymentFundingReaction   -> RecordFunding         -> PaymentFailed
PaymentFunded                                      -> PaymentChecksGateway     -> POST /checks of the fraud service, POST /screenings of the sanctions service
POST /api/webhooks/fraud-checks                    -> RecordFraudResult        -> FraudCheckPassed or FraudCheckFailed, then PaymentCleared or PaymentRefused
POST /api/webhooks/sanctions-screenings            -> RecordScreeningResult    -> SanctionsScreeningPassed or SanctionsScreeningFailed, then PaymentCleared or PaymentRefused
PaymentCleared                                     -> PaymentSubmissionGateway -> POST /payments of the payment network, then RecordSubmission -> PaymentSubmitted
POST /api/webhooks/payment-network                 -> RecordSettlement         -> PaymentSettled
POST /api/webhooks/payment-network                 -> RecordReturn             -> PaymentReturned
PaymentSettled                                     -> PaymentHoldsReaction     -> CaptureHold           -> HoldCaptured
PaymentRefused, PaymentFailed or PaymentReturned   -> PaymentHoldsReaction     -> ReleaseHold           -> HoldReleased
the payment events                                 -> PaymentSummaryView       (GET /api/payments/{id})
```

The deadlines:

```
each second                           -> PaymentTimeoutsScheduler reads the passed deadlines in PaymentDeadlinesView
the deadline of the checks passed     -> TimeOutChecks     -> PaymentChecksRetried     -> PaymentChecksGateway,
                                                                                          or PaymentFailed after the last attempt
the deadline of the submission passed -> TimeOutSubmission -> PaymentSubmissionRetried -> PaymentSubmissionGateway,
                                                                                          or PaymentEscalated after the last attempt
the deadline of the settlement passed -> TimeOutSettlement -> PaymentStatusQueried     -> PaymentStatusGateway -> RecordSettlement or RecordReturn,
                                                                                          or PaymentEscalated after the last attempt
```

Each stage waits 10 seconds by default (`payment.DefaultTimeouts`). A payment asks 3 times for one stage
(`payment.MaxAttempts`), and then it fails or escalates.

### Stages of the Payment

The payment aggregate keeps the stage of the payment. A command acts only in the stages of this table. In
the other stages, the command records nothing.

| Stage | After | The payment waits for | Commands that act |
| --- | --- | --- | --- |
| `requested` | `PaymentRequested` | The answer of the account to the hold | `RecordFunding` |
| `funded` | `PaymentFunded` | The fraud check and the sanctions screening | `RecordFraudResult`, `RecordScreeningResult`, `TimeOutChecks` |
| `cleared` | `PaymentCleared` | The payment network to take the payment | `RecordSubmission`, `TimeOutSubmission`, `RecordSettlement`, `RecordReturn` |
| `submitted` | `PaymentSubmitted` | The payment network to settle or return the payment | `RecordSettlement`, `RecordReturn`, `TimeOutSettlement` |
| `escalated` | `PaymentEscalated` | A person, and a webhook of the payment network | `RecordSettlement`, `RecordReturn` |
| `settled`, `returned`, `refused`, `failed` | `PaymentSettled`, `PaymentReturned`, `PaymentRefused`, `PaymentFailed` | Nothing. The payment ended | None |

- The payment records `PaymentCleared` when both checks passed, and `PaymentRefused` at the first check
  that fails.
- `RecordSettlement` and `RecordReturn` act from the time the payment clears. The network can settle a
  payment before `PaymentSubmissionGateway` records the reference of the network.
- After the network can have the payment, the payment does not fail. It escalates to a person and keeps
  the hold, because the network can settle the payment late.

### Events

| Event | Stream | Description | Key data |
| --- | --- | --- | --- |
| `HoldPlaced` | `account-<id>` | The account holds the amount of a payment. The amount stays in the balance, but it is not available | `accountId`, `paymentId`, `amount` |
| `HoldRejected` | `account-<id>` | The account cannot hold the amount of a payment | `accountId`, `paymentId`, `amount`, `reason` |
| `HoldReleased` | `account-<id>` | The held amount of a payment is available again | `accountId`, `paymentId`, `amount` |
| `HoldCaptured` | `account-<id>` | The held amount of a payment left the account | `accountId`, `paymentId`, `amount` |
| `PaymentRequested` | `payment-<id>` | A client asked to pay an amount from an account to an account at another bank | `paymentId`, `accountId`, `creditorIban`, `amount` |
| `PaymentFunded` | `payment-<id>` | The account holds the amount. The checks start | `paymentId`, `checksDeadline` |
| `PaymentFailed` | `payment-<id>` | The payment ended before the payment network got it | `paymentId`, `accountId`, `reason` |
| `FraudCheckPassed` | `payment-<id>` | The fraud service found no risk of fraud | `paymentId` |
| `FraudCheckFailed` | `payment-<id>` | The fraud service found a risk of fraud | `paymentId`, `reason` |
| `SanctionsScreeningPassed` | `payment-<id>` | The sanctions service found the creditor on no sanctions list | `paymentId` |
| `SanctionsScreeningFailed` | `payment-<id>` | The sanctions service found the creditor on a sanctions list | `paymentId`, `reason` |
| `PaymentCleared` | `payment-<id>` | Both checks passed | `paymentId`, `submissionDeadline` |
| `PaymentRefused` | `payment-<id>` | A check failed, so the bank refuses the payment | `paymentId`, `accountId`, `reason` |
| `PaymentChecksRetried` | `payment-<id>` | The payment asks for the checks again | `paymentId`, `attempt`, `checksDeadline` |
| `PaymentSubmitted` | `payment-<id>` | The payment network took the payment | `paymentId`, `networkReference`, `settlementDeadline` |
| `PaymentSubmissionRetried` | `payment-<id>` | The payment asks to go to the payment network again | `paymentId`, `attempt`, `submissionDeadline` |
| `PaymentStatusQueried` | `payment-<id>` | The payment asks the payment network for its status | `paymentId`, `attempt`, `settlementDeadline` |
| `PaymentEscalated` | `payment-<id>` | The payment network did not answer, so a person must check the payment | `paymentId`, `reason` |
| `PaymentSettled` | `payment-<id>` | The payment network settled the payment, so the money left the bank | `paymentId`, `accountId` |
| `PaymentReturned` | `payment-<id>` | The other bank returned the payment | `paymentId`, `accountId`, `reason` |

- The store records the time of each event. A payload carries a time only for a deadline.
- `PaymentFailed`, `PaymentRefused`, `PaymentReturned` and `PaymentSettled` carry `accountId`.
  Therefore, `PaymentHoldsReaction` sends the hold command with no lookup.

### Commands

| Command | Actor | Produces | Refused when | Data |
| --- | --- | --- | --- | --- |
| `PlaceHold` | `PaymentFundingReaction` | `HoldPlaced` or `HoldRejected` | `AccountExists` | `accountId`, `paymentId`, `amount` ← `PaymentRequested` |
| `ReleaseHold` | `PaymentHoldsReaction` | `HoldReleased` | `AccountExists` | `accountId`, `paymentId` ← `PaymentRefused`, `PaymentFailed` or `PaymentReturned` |
| `CaptureHold` | `PaymentHoldsReaction` | `HoldCaptured` | `AccountExists`, `HoldExists` | `accountId`, `paymentId` ← `PaymentSettled` |
| `RequestPayment` | Client | `PaymentRequested` | `RequestedOnce`, `PositiveAmount`, `HasCreditor`, `AccountExists` | `paymentId`: made by the API. `accountId`, `creditorIban`, `amount`: the body |
| `RecordFunding` | `PaymentFundingReaction` | `PaymentFunded` or `PaymentFailed` | `PaymentExists` | `paymentId` ← `HoldPlaced` or `HoldRejected`. `held`: `true` after `HoldPlaced`, `false` after `HoldRejected`. `reason` ← `HoldRejected.reason` |
| `RecordFraudResult` | Fraud service | `FraudCheckPassed` or `FraudCheckFailed`, then `PaymentCleared` or `PaymentRefused` | `PaymentExists` | `paymentId`, `passed`, `reason`: the body of the webhook |
| `RecordScreeningResult` | Sanctions service | `SanctionsScreeningPassed` or `SanctionsScreeningFailed`, then `PaymentCleared` or `PaymentRefused` | `PaymentExists` | `paymentId`, `passed`, `reason`: the body of the webhook |
| `TimeOutChecks` | `PaymentTimeoutsScheduler` | `PaymentChecksRetried`, or `PaymentFailed` after the last attempt | `PaymentExists` | `paymentId`, `attempt` ← `PaymentDeadlinesView` |
| `RecordSubmission` | `PaymentSubmissionGateway` | `PaymentSubmitted` | `PaymentExists` | `paymentId` ← `PaymentCleared` or `PaymentSubmissionRetried`. `networkReference` ← the answer of the payment network |
| `TimeOutSubmission` | `PaymentTimeoutsScheduler` | `PaymentSubmissionRetried`, or `PaymentEscalated` after the last attempt | `PaymentExists` | `paymentId`, `attempt` ← `PaymentDeadlinesView` |
| `TimeOutSettlement` | `PaymentTimeoutsScheduler` | `PaymentStatusQueried`, or `PaymentEscalated` after the last attempt | `PaymentExists` | `paymentId`, `attempt` ← `PaymentDeadlinesView` |
| `RecordSettlement` | Payment network, `PaymentStatusGateway` | `PaymentSettled` | `PaymentExists` | `paymentId`: the body of the webhook, or `PaymentStatusQueried` |
| `RecordReturn` | Payment network, `PaymentStatusGateway` | `PaymentReturned` | `PaymentExists` | `paymentId`, `reason`: the body of the webhook, or `PaymentStatusQueried` and the status answer of the network |

- The Go type of `RequestPayment` is `command.Request` in `internal/use_cases/payment/command`. The other
  commands have the same names in the model and in the code.
- No command carries a deadline. The handler adds the timeout of the next stage to the current time,
  and the payment records the deadline in its event.
- Each command has a fixed ID. A reaction or a gateway makes the ID with `dispatch.CommandFor`. The
  scheduler makes the ID from the payment ID, the stage and the attempt. Therefore, a tick that runs
  again sends a command with the same ID, and the store records the command once.

### Rules

| Rule | Aggregate | Statement | Code |
| --- | --- | --- | --- |
| `RequestedOnce` | Payment | A payment is requested once | `payment.ErrAlreadyRequested` |
| `PositiveAmount` | Payment | An amount is more than 0 | `payment.ErrInvalidAmount` |
| `HasCreditor` | Payment | A payment has a creditor IBAN | `payment.ErrNoCreditor` |
| `AccountExists` | Payment | The account of a payment was opened | `payment.ErrUnknownAccount`, in the handler of `RequestPayment` |
| `PaymentExists` | Payment | A command changes only a payment that was requested | `payment.ErrNotRequested` |
| `AccountExists` | Account | A command changes only an account that was opened | `account.ErrNotOpened` |
| `EnoughFunds` | Account | A withdrawal is at most the available balance: the balance minus the holds | `account.ErrInsufficientFunds` |
| `HoldExists` | Account | The account captures only a hold that it placed and did not release | `account.ErrNoHold` |

The other rules of the account are the same as in [event-model-account.md](event-model-account.md).

The payment checks `AccountExists` before the hold. The account refuses a hold on an account that was not
opened, and a refusal records no event. Without the rule, the payment waits for the hold with no end. A
closed account passes the rule: the account then rejects the hold, and the payment fails.

A failure that the process acts on is an event, not a refusal:

- `HoldRejected`: the account is closed, the amount is not more than 0, or the available balance is less
  than the amount.
- `FraudCheckFailed` or `SanctionsScreeningFailed`, then `PaymentRefused`: a check found a risk.
- `PaymentFailed`: the account rejected the hold, or the checks did not answer after the last attempt.
- `PaymentReturned`: the other bank sent the payment back.
- `PaymentEscalated`: the payment network did not take or did not settle the payment after the last
  attempt.

### Commands That Record Nothing

Some commands succeed and record nothing. These are not refusals, and a webhook then gets `200 OK`. The
specs show them with `then = []`. The design of the payment names three of these cases:

| Name | Statement | Code |
| --- | --- | --- |
| `StageMatches` | A command acts only in its stages, as [Stages of the Payment](#stages-of-the-payment) shows. A timeout acts only for the attempt that is due | The check of the stage and of the attempt at the start of each method of `payment.Payment` |
| `ResultOnce` | Each check records one result. A result that arrives again records nothing | `fraudPassed` and `screeningPassed` of `payment.Payment` |
| `EndsOnce` | A payment ends once: settled, returned, refused or failed | No command acts in an end stage |

The account also records nothing in these cases:

- `PlaceHold` for a payment that the account decided.
- `ReleaseHold` when the account holds nothing for the payment. The account placed no hold, rejected the
  hold, or released or captured the hold before.
- `CaptureHold` for a hold that the account captured.

### Views

| View | Purpose | Backs | Data lineage |
| --- | --- | --- | --- |
| `AccountBalanceView` | One account with its owner, balance and status | `GET /api/accounts/{id}` | The lineage of [event-model-account.md](event-model-account.md), plus `balance` ← `- HoldCaptured.amount`. A placed or released hold does not change the balance |
| `PaymentSummaryView` | One payment with its status | `GET /api/payments/{id}` | `id`, `accountId`, `creditorIban`, `amount` ← `PaymentRequested`. `status` ← `requested` after `PaymentRequested`, `funded` after `PaymentFunded`, and the same for `failed`, `cleared`, `refused`, `submitted`, `escalated`, `settled` and `returned`. `reason` ← the reason of `PaymentFailed`, `PaymentRefused`, `PaymentEscalated` or `PaymentReturned`, and empty after the other events. `networkReference` ← `PaymentSubmitted.networkReference`. `version` ← the stream version of the last event |
| `PaymentDeadlinesView` | The todo list of `PaymentTimeoutsScheduler`: each open payment with the stage that waits, the attempt and the deadline | `PaymentTimeoutsScheduler` | The next table |

The lineage of `PaymentDeadlinesView`:

| Event | `stage` | `attempt` | `deadline` |
| --- | --- | --- | --- |
| `PaymentFunded` | `checks` | 1 | `checksDeadline` |
| `PaymentChecksRetried` | `checks` | `attempt` | `checksDeadline` |
| `PaymentCleared` | `submission` | 1 | `submissionDeadline` |
| `PaymentSubmissionRetried` | `submission` | `attempt` | `submissionDeadline` |
| `PaymentSubmitted` | `settlement` | 1 | `settlementDeadline` |
| `PaymentStatusQueried` | `settlement` | `attempt` | `settlementDeadline` |
| `PaymentFailed`, `PaymentRefused`, `PaymentEscalated`, `PaymentSettled`, `PaymentReturned` | The projection removes the row | | |

- `paymentId` ← the event. `version` ← the stream version of the last event.
- An escalated payment has no row, so no deadline asks the network again. A person checks the payment,
  and a later webhook of the payment network ends it.

### Automations

| Trigger | Automation | Consumer | Command | Result event |
| --- | --- | --- | --- | --- |
| `PaymentRequested` | `PaymentFundingReaction` | `payment-funding` | `PlaceHold` | `HoldPlaced` or `HoldRejected` |
| `HoldPlaced`, `HoldRejected` | `PaymentFundingReaction` | `payment-funding` | `RecordFunding` | `PaymentFunded` or `PaymentFailed` |
| `PaymentFunded`, `PaymentChecksRetried` | `PaymentChecksGateway` | `payment-checks` | None. It calls `POST /checks` of the fraud service and `POST /screenings` of the sanctions service | None. The webhooks send `RecordFraudResult` and `RecordScreeningResult` |
| `PaymentCleared`, `PaymentSubmissionRetried` | `PaymentSubmissionGateway` | `payment-submission` | `RecordSubmission`, after `POST /payments` of the payment network | `PaymentSubmitted` |
| `PaymentStatusQueried` | `PaymentStatusGateway` | `payment-status` | `RecordSettlement` or `RecordReturn`, after `GET /payments/{reference}` of the payment network. None for another status | `PaymentSettled` or `PaymentReturned` |
| Each second | `PaymentTimeoutsScheduler` | `payment-timeouts` | `TimeOutChecks`, `TimeOutSubmission` or `TimeOutSettlement`, from the `stage` of each passed deadline in `PaymentDeadlinesView` | The events of the command |
| `PaymentSettled` | `PaymentHoldsReaction` | `payment-holds` | `CaptureHold` | `HoldCaptured` |
| `PaymentRefused`, `PaymentFailed`, `PaymentReturned` | `PaymentHoldsReaction` | `payment-holds` | `ReleaseHold` | `HoldReleased`, or nothing when the account holds nothing for the payment |

The gateways:

- `PaymentChecksGateway` and `PaymentSubmissionGateway` read the amount and the creditor IBAN from
  `PaymentRequested` in the stream of the payment. `PaymentStatusGateway` reads the network reference
  from `PaymentSubmitted`. The aggregate does not keep these values, because no rule reads them.
- A call to an outside system carries an `Idempotency-Key`. A check gets a new key for each attempt, so a
  check service that lost a request does the check again. The submission keeps one key for all attempts,
  so the payment network takes the payment once.
- `outside.Client` tries a call again after a network error or a `5xx` answer, at most 3 times. When the
  call still fails, the gateway logs the error and does not stop its consumer. The next deadline of the
  payment asks again.

### Message Processing Patterns

| # | From | To | Pattern | Component |
| --- | --- | --- | --- | --- |
| 1 | `POST /api/payments` | `RequestPayment` | UI trigger | `internal/api/use_cases/payment` |
| 2 | `RequestPayment` | `PaymentRequested` | Aggregate | `internal/use_cases/payment/request` |
| 3 | `PaymentRequested` | `PlaceHold` | Reactor | `funding.Reaction`, consumer `payment-funding` |
| 4 | `PlaceHold` | `HoldPlaced`, `HoldRejected` | Aggregate | `internal/use_cases/account/holds` |
| 5 | `HoldPlaced`, `HoldRejected` | `RecordFunding` | Reactor | `funding.Reaction`, consumer `payment-funding` |
| 6 | `RecordFunding` | `PaymentFunded`, `PaymentFailed` | Aggregate | `internal/use_cases/payment/record` |
| 7 | `PaymentFunded`, `PaymentChecksRetried` | The fraud service and the sanctions service | Gateway | `checks.Gateway`, consumer `payment-checks` |
| 8 | `POST /api/webhooks/fraud-checks` | `RecordFraudResult` | Translation | `internal/api/use_cases/webhook` |
| 9 | `POST /api/webhooks/sanctions-screenings` | `RecordScreeningResult` | Translation | `internal/api/use_cases/webhook` |
| 10 | `RecordFraudResult`, `RecordScreeningResult` | The four check events, `PaymentCleared`, `PaymentRefused` | Aggregate | `internal/use_cases/payment/record` |
| 11 | `PaymentCleared`, `PaymentSubmissionRetried` | The payment network, then `RecordSubmission` | Gateway | `submission.Gateway`, consumer `payment-submission` |
| 12 | `RecordSubmission` | `PaymentSubmitted` | Aggregate | `internal/use_cases/payment/record` |
| 13 | `POST /api/webhooks/payment-network` | `RecordSettlement`, `RecordReturn` | Translation | `internal/api/use_cases/webhook` |
| 14 | `PaymentStatusQueried` | The payment network, then `RecordSettlement` or `RecordReturn` | Gateway | `status.Gateway`, consumer `payment-status` |
| 15 | `RecordSettlement`, `RecordReturn` | `PaymentSettled`, `PaymentReturned` | Aggregate | `internal/use_cases/payment/record` |
| 16 | Eleven payment events | `PaymentDeadlinesView` | Projection | `deadlines.Projection`, consumer `payment-deadlines` |
| 17 | `PaymentDeadlinesView`, each second | `TimeOutChecks`, `TimeOutSubmission`, `TimeOutSettlement` | Scheduler | `deadlines.Reporter`, worker `payment-timeouts` |
| 18 | `TimeOutChecks`, `TimeOutSubmission`, `TimeOutSettlement` | `PaymentChecksRetried`, `PaymentSubmissionRetried`, `PaymentStatusQueried`, `PaymentFailed`, `PaymentEscalated` | Aggregate | `internal/use_cases/payment/timeout` |
| 19 | `PaymentSettled`, `PaymentRefused`, `PaymentFailed`, `PaymentReturned` | `CaptureHold`, `ReleaseHold` | Reactor | `holds.Reaction`, consumer `payment-holds` |
| 20 | `CaptureHold`, `ReleaseHold` | `HoldCaptured`, `HoldReleased` | Aggregate | `internal/use_cases/account/holds` |
| 21 | Nine payment events | `PaymentSummaryView` | Projection | `summary.Projection`, consumer `payments` |
| 22 | `HoldCaptured` | `AccountBalanceView` | Projection | `balance.Projection`, consumer `account-balances` |

- The payment aggregate is `payment.Payment` in `internal/use_cases/payment/payment.go`. The holds of the
  account are in `internal/use_cases/account/hold.go`.
- The `catchup` process runs the reactions, the gateways, the projections and the scheduler. Each one
  runs under a lock on its name. Therefore, the scheduler has a name that is different from the name
  of the projection.
- The reactions and the gateways of the payment start with the first event in the store.

## Where the Model and the Code Are Different

The model shows the process manager with the usual Event Modeling elements. The code has the same steps,
with these differences:

- **`PaymentChecksGateway` has no command.** In Event Modeling, a gateway sends a command that records no
  event. The code has no such command, because the gateway calls the fraud service and the sanctions
  service itself. The model does not add a command that the code does not have. The answers come back
  through the translations of `POST /api/webhooks/fraud-checks` and `POST /api/webhooks/sanctions-screenings`.
- **The slice "Ask for Checks" has no spec.** emod accepts a spec with `then = []` only in a slice with a
  command or a translation. The tests of `checks.Gateway` show the calls.
- **The descriptions tell the calls of the other two gateways.** `PaymentSubmissionGateway` calls the
  payment network before it sends `RecordSubmission`. `PaymentStatusGateway` calls the payment network
  before it sends `RecordSettlement` or `RecordReturn`. emod has no element for a call inside an
  automation.
- **The gateways read a stream, not a view.** They read `PaymentRequested` and `PaymentSubmitted` from the
  stream of the payment. emod has no element for the read of a stream, so the gateways have no `reads`.
- **The route of each check is two translations.** One route sends one command, and the payment records a
  pass or a failure. emod reports a command with three or more `command -> event` lines as the error
  `left-chair`. `RecordFraudResult` has four results: the result of the check, then `PaymentCleared` or
  `PaymentRefused`. Therefore, each translation holds one result of the check, and the flow holds
  `PaymentCleared` and `PaymentRefused`. `RecordScreeningResult` has the same form.
- **The route of the payment network is two translations.** The route sends `RecordSettlement` or
  `RecordReturn` from the status in the body. An emod translation has one command.
- **`GET /api/payments/{id}` is a trigger with no command.** emod needs a reader for each view. The client
  reads the payment and sends no command after the read.
- **The commands that record nothing are not rules in the model.** In emod, an invariant refuses a
  command. `StageMatches`, `ResultOnce` and `EndsOnce` refuse nothing: the command succeeds and records
  nothing. Therefore, the model shows them only in specs with `then = []`.
- **`AccountExists` of the payment reads another stream.** The handler of `RequestPayment` loads the
  account stream before the payment records an event. A spec can show only the events of its own
  aggregate, so its spec has no account history.
- **A spec cannot show time.** Each spec of `PaymentTimeoutsScheduler` gives the event that starts the
  deadline, and the name of the spec tells that the deadline passed.
- **One component is more than one automation.** An emod automation has one trigger and one command.
  Therefore, the model shows each reaction and each gateway of the payment, and the scheduler, more than
  once, with the same name. In the code, each one is one consumer or one worker.
- **Two payment reactions are in the Accounts context.** emod puts each automation in the slice of the
  command that it sends. Therefore, `PaymentHoldsReaction` and the `PaymentFundingReaction` that sends
  `PlaceHold` are in the slices of the hold commands. In the code, they are in
  `internal/use_cases/payment/funding` and `internal/use_cases/payment/holds`.

## Autonomous Components

| Component | Owned events | Owned commands | Consumes events from |
| --- | --- | --- | --- |
| Accounts | The account events of [event-model-account.md](event-model-account.md), plus `HoldPlaced`, `HoldRejected`, `HoldReleased`, `HoldCaptured` | The account commands, plus `PlaceHold`, `ReleaseHold`, `CaptureHold` | None. The payment reactions send the hold commands |
| Payments | The 16 payment events | The 10 payment commands | `HoldPlaced` and `HoldRejected` of Accounts. The answers of the outside systems come through the webhook routes |

## Implementation Slices

The slices of [event-model-account.md](event-model-account.md) come first. In the slice View Account
Balance, `AccountBalanceView` also subscribes to `HoldCaptured`. This table lists the slices that the
payment adds.

| # | Slice | Events | Commands | Views | Package |
| --- | --- | --- | --- | --- | --- |
| 1 | Place Hold | `HoldPlaced`, `HoldRejected` | `PlaceHold` | | `internal/use_cases/account/holds`, `internal/use_cases/payment/funding` |
| 2 | Release Hold | `HoldReleased` | `ReleaseHold` | | `internal/use_cases/account/holds`, `internal/use_cases/payment/holds` |
| 3 | Capture Hold | `HoldCaptured` | `CaptureHold` | | `internal/use_cases/account/holds`, `internal/use_cases/payment/holds` |
| 4 | Request Payment | `PaymentRequested` | `RequestPayment` | | `internal/use_cases/payment/request` |
| 5 | Record Funding | `PaymentFunded`, `PaymentFailed` | `RecordFunding` | | `internal/use_cases/payment/record`, `internal/use_cases/payment/funding` |
| 6 | Ask for Checks | | | | `internal/use_cases/payment/checks` |
| 7 | Record Fraud Result | `FraudCheckPassed`, `FraudCheckFailed`, `PaymentCleared`, `PaymentRefused` | `RecordFraudResult` | | `internal/use_cases/payment/record`, `internal/api/use_cases/webhook` |
| 8 | Record Screening Result | `SanctionsScreeningPassed`, `SanctionsScreeningFailed`, `PaymentCleared`, `PaymentRefused` | `RecordScreeningResult` | | `internal/use_cases/payment/record`, `internal/api/use_cases/webhook` |
| 9 | Time Out Checks | `PaymentChecksRetried`, `PaymentFailed` | `TimeOutChecks` | | `internal/use_cases/payment/timeout`, `internal/use_cases/payment/deadlines` |
| 10 | Record Submission | `PaymentSubmitted` | `RecordSubmission` | | `internal/use_cases/payment/record`, `internal/use_cases/payment/submission` |
| 11 | Time Out Submission | `PaymentSubmissionRetried`, `PaymentEscalated` | `TimeOutSubmission` | | `internal/use_cases/payment/timeout`, `internal/use_cases/payment/deadlines` |
| 12 | Time Out Settlement | `PaymentStatusQueried`, `PaymentEscalated` | `TimeOutSettlement` | | `internal/use_cases/payment/timeout`, `internal/use_cases/payment/deadlines` |
| 13 | Record Settlement | `PaymentSettled` | `RecordSettlement` | | `internal/use_cases/payment/record`, `internal/use_cases/payment/status`, `internal/api/use_cases/webhook` |
| 14 | Record Return | `PaymentReturned` | `RecordReturn` | | `internal/use_cases/payment/record`, `internal/use_cases/payment/status`, `internal/api/use_cases/webhook` |
| 15 | View Payment Summary | | | `PaymentSummaryView` | `internal/use_cases/payment/summary`, `internal/api/use_cases/payment` |
| 16 | View Payment Deadlines | | | `PaymentDeadlinesView` | `internal/use_cases/payment/deadlines` |

## Lint Warnings

The number after the name of an automation is the number of automations with that name in the model.

| Rule | Element | Why the model keeps it |
| --- | --- | --- |
| `automation/missing-todo-list` | `WelcomeBonusReaction` | The same as in [event-model-account.md](event-model-account.md#lint-warnings) |
| `automation/missing-todo-list` | `PaymentFundingReaction` (3), `PaymentHoldsReaction` (4) | The reactions read no view. Each one keeps a checkpoint, each command has a fixed ID, and each aggregate records a step once. Therefore, no list of work is necessary |
| `automation/missing-todo-list` | `PaymentChecksGateway` (2), `PaymentSubmissionGateway` (2), `PaymentStatusGateway` (2) | The gateways read the stream of the payment, not a view. When a call fails, the list of work is `PaymentDeadlinesView`: the deadline of the stage passes, and the payment asks again. The code has no view for each gateway |
| `clickbait-event` | `AccountClosed` | The same as in [event-model-account.md](event-model-account.md#lint-warnings) |
| `clickbait-event` | `FraudCheckPassed`, `SanctionsScreeningPassed` | The name tells exactly what happened. The payment needs only the fact that the check passed, and it clears when both checks passed. A failed check carries its reason |
| `god-view` | `AccountBalanceView` | A balance must include each money movement of the account. With the holds, that is 6 events, and each is a different business fact |
| `god-view` | `PaymentSummaryView` | The status must follow each stage of the payment. That is 9 events, and each is a different business fact. One `PaymentStatusChanged` event in their place is the State Obsession anti-pattern |
| `god-view` | `PaymentDeadlinesView` | A todo list must start a deadline at each stage and each attempt, and remove the deadline at each end of the payment. That is 11 events |
