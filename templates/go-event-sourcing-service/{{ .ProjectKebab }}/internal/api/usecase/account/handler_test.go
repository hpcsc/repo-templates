//go:build unit

package account_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/response"
	apiTest "github.com/hpcsc/{{ .ProjectKebab }}/internal/api/test"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/usecase/account"
	commonTest "github.com/hpcsc/{{ .ProjectKebab }}/internal/common/test"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command/fake"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/balance"
	accountCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/command"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	t.Run("open", func(t *testing.T) {
		t.Run("dispatches an open command for the owner and returns the new account ID", func(t *testing.T) {
			handler := fake.NewHandler[accountCommand.Open]()
			bus := command.NewBus()
			bus.Register[accountCommand.Open](handler)
			router := newRouter(bus, nil)

			w := send(router, "POST", "/accounts", `{"owner": "alice"}`)

			require.Equal(t, http.StatusOK, w.Code)
			payload := handler.TriggeredWithCommand().Payload
			require.Equal(t, "alice", payload.Owner)
			body := commonTest.UnmarshalJson[response.Response](t, w.Body.Bytes())
			require.Equal(t, payload.AccountID.String(), body.Data.(map[string]any)["id"])
		})

		t.Run("rejects a body without an owner", func(t *testing.T) {
			bus := command.NewBus()
			bus.Register[accountCommand.Open](fake.NewHandler[accountCommand.Open]())
			router := newRouter(bus, nil)

			w := send(router, "POST", "/accounts", `{}`)

			require.Equal(t, http.StatusBadRequest, w.Code)
		})
	})

	t.Run("deposit", func(t *testing.T) {
		t.Run("dispatches a deposit command for the account in the path", func(t *testing.T) {
			handler := fake.NewHandler[accountCommand.Deposit]()
			bus := command.NewBus()
			bus.Register[accountCommand.Deposit](handler)
			router := newRouter(bus, nil)
			id := uuid.New()

			w := send(router, "POST", fmt.Sprintf("/accounts/%s/deposits", id), `{"amount": 500}`)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, accountCommand.Deposit{AccountID: id, Amount: 500}, handler.TriggeredWithCommand().Payload)
		})

		t.Run("rejects an account ID that is not a UUID", func(t *testing.T) {
			bus := command.NewBus()
			bus.Register[accountCommand.Deposit](fake.NewHandler[accountCommand.Deposit]())
			router := newRouter(bus, nil)

			w := send(router, "POST", "/accounts/not-a-uuid/deposits", `{"amount": 500}`)

			require.Equal(t, http.StatusBadRequest, w.Code)
			apiTest.RequireErrorResponse(t, w, "not a UUID")
		})

		t.Run("returns a conflict when the account changed at the same time", func(t *testing.T) {
			handler := fake.NewHandler[accountCommand.Deposit]().WithError(fmt.Errorf("%w: stream moved", event.ErrConcurrencyConflict))
			bus := command.NewBus()
			bus.Register[accountCommand.Deposit](handler)
			router := newRouter(bus, nil)

			w := send(router, "POST", fmt.Sprintf("/accounts/%s/deposits", uuid.New()), `{"amount": 500}`)

			require.Equal(t, http.StatusConflict, w.Code)
		})
	})

	t.Run("close", func(t *testing.T) {
		t.Run("dispatches a close command for the account in the path", func(t *testing.T) {
			handler := fake.NewHandler[accountCommand.Close]()
			bus := command.NewBus()
			bus.Register[accountCommand.Close](handler)
			router := newRouter(bus, nil)
			id := uuid.New()

			w := send(router, "POST", fmt.Sprintf("/accounts/%s/close", id), "")

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, accountCommand.Close{AccountID: id}, handler.TriggeredWithCommand().Payload)
		})
	})

	t.Run("get", func(t *testing.T) {
		t.Run("returns the balance from the read model", func(t *testing.T) {
			id := uuid.NewString()
			router := newRouter(command.NewBus(), &balance.Balance{ID: id, Owner: "alice", Balance: 1300, Version: 3})

			w := send(router, "GET", "/accounts/"+id, "")

			require.Equal(t, http.StatusOK, w.Code)
			require.Contains(t, w.Body.String(), `"balance":1300`)
		})

		t.Run("returns not found when the read model has no row for the account", func(t *testing.T) {
			router := newRouter(command.NewBus(), nil)

			w := send(router, "GET", "/accounts/"+uuid.NewString(), "")

			require.Equal(t, http.StatusNotFound, w.Code)
		})
	})
}

type fakeBalances struct {
	balance *balance.Balance
}

func (f *fakeBalances) ByID(context.Context, string) (*balance.Balance, error) {
	return f.balance, nil
}

func newRouter(bus *command.Bus, b *balance.Balance) http.Handler {
	return apiTest.NewRouterWithRoutable(account.NewHandler(bus, &fakeBalances{balance: b}))
}

func send(router http.Handler, method string, path string, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	return w
}
