//go:build unit

package transfer_test

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
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/usecase/transfer"
	commonTest "github.com/hpcsc/{{ .ProjectKebab }}/internal/common/test"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command/fake"
	transferCommand "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/transfer/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/transfer/process"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	t.Run("start", func(t *testing.T) {
		t.Run("dispatches a start command and returns the new transfer ID", func(t *testing.T) {
			handler := fake.NewHandler[transferCommand.Start]()
			bus := command.NewBus()
			bus.Register[transferCommand.Start](handler)
			from, to := uuid.New(), uuid.New()

			w := send(newRouter(bus, nil), "POST", "/transfers", fmt.Sprintf(`{"from": %q, "to": %q, "amount": 300}`, from, to))

			require.Equal(t, http.StatusOK, w.Code)
			payload := handler.TriggeredWithCommand().Payload
			require.Equal(t, from, payload.From)
			require.Equal(t, to, payload.To)
			require.Equal(t, int64(300), payload.Amount)
			body := commonTest.UnmarshalJson[response.Response](t, w.Body.Bytes())
			require.Equal(t, payload.TransferID.String(), body.Data.(map[string]any)["id"])
		})

		t.Run("rejects an account ID that is not a UUID", func(t *testing.T) {
			bus := command.NewBus()
			bus.Register[transferCommand.Start](fake.NewHandler[transferCommand.Start]())

			w := send(newRouter(bus, nil), "POST", "/transfers", fmt.Sprintf(`{"from": "not-a-uuid", "to": %q, "amount": 300}`, uuid.New()))

			require.Equal(t, http.StatusBadRequest, w.Code)
			apiTest.RequireErrorResponse(t, w, "must be account IDs")
		})
	})

	t.Run("get", func(t *testing.T) {
		t.Run("returns the state of the transfer process", func(t *testing.T) {
			id := uuid.NewString()

			w := send(newRouter(command.NewBus(), &process.Status{ID: id, State: "failed", Reason: "account is closed"}), "GET", "/transfers/"+id, "")

			require.Equal(t, http.StatusOK, w.Code)
			require.Contains(t, w.Body.String(), `"state":"failed"`)
			require.Contains(t, w.Body.String(), `"reason":"account is closed"`)
		})

		t.Run("returns not found when the process has no row for the transfer", func(t *testing.T) {
			w := send(newRouter(command.NewBus(), nil), "GET", "/transfers/"+uuid.NewString(), "")

			require.Equal(t, http.StatusNotFound, w.Code)
		})
	})
}

type fakeStatuses struct {
	status *process.Status
}

func (f *fakeStatuses) ByID(context.Context, string) (*process.Status, error) {
	return f.status, nil
}

func newRouter(bus *command.Bus, status *process.Status) http.Handler {
	return apiTest.NewRouterWithRoutable(transfer.NewHandler(bus, &fakeStatuses{status: status}))
}

func send(router http.Handler, method string, path string, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	return w
}
