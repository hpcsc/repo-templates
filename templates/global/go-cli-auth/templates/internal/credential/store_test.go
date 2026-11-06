//go:build unit

package credential_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/hpcsc/go-cli-project/internal/credential"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

func newStore(t *testing.T, mode string) credential.Store {
	t.Helper()
	t.Setenv("GO_CLI_PROJECT_CREDENTIAL_STORE", mode)
	store, err := credential.NewStore()
	require.NoError(t, err)
	return store
}

func TestStore(t *testing.T) {
	t.Run("set", func(t *testing.T) {
		t.Run("keeps the value in the OS credential store and writes no file", func(t *testing.T) {
			dir := isolate(t)
			keyring.MockInit()
			store := newStore(t, "keyring")

			source, err := store.Set("api.example.test", "secret")

			require.NoError(t, err)
			require.Equal(t, "the OS credential store", source)
			require.NoFileExists(t, filepath.Join(dir, "credentials.json"))
			found, err := store.Get("api.example.test")
			require.NoError(t, err)
			require.Equal(t, "secret", found.Value)
			require.Equal(t, "the OS credential store", found.Source)
		})

		t.Run("writes the value to a file that only the user can read when the OS credential store fails", func(t *testing.T) {
			file := filepath.Join(isolate(t), "credentials.json")
			keyring.MockInitWithError(errors.New("no secret service"))
			store := newStore(t, "keyring")

			source, err := store.Set("api.example.test", "secret")

			require.NoError(t, err)
			require.Equal(t, file, source)
			info, err := os.Stat(file)
			require.NoError(t, err)
			require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
			found, err := store.Get("api.example.test")
			require.NoError(t, err)
			require.Equal(t, "secret", found.Value)
			require.Equal(t, file, found.Source)
		})

		t.Run("writes only to the file when GO_CLI_PROJECT_CREDENTIAL_STORE is file", func(t *testing.T) {
			file := filepath.Join(isolate(t), "credentials.json")
			keyring.MockInit()

			_, err := newStore(t, "file").Set("api.example.test", "secret")

			require.NoError(t, err)
			found, err := newStore(t, "keyring").Get("api.example.test")
			require.NoError(t, err)
			require.Equal(t, file, found.Source)
		})

		t.Run("removes the copy in the file when the OS credential store works again", func(t *testing.T) {
			file := filepath.Join(isolate(t), "credentials.json")
			keyring.MockInitWithError(errors.New("locked"))
			_, err := newStore(t, "keyring").Set("api.example.test", "old")
			require.NoError(t, err)
			require.FileExists(t, file)
			keyring.MockInit()

			_, err = newStore(t, "keyring").Set("api.example.test", "new")

			require.NoError(t, err)
			require.NoFileExists(t, file)
		})
	})

	t.Run("get", func(t *testing.T) {
		t.Run("a host with no stored value is ErrNotStored", func(t *testing.T) {
			isolate(t)

			_, err := newStore(t, "file").Get("api.example.test")

			require.ErrorIs(t, err, credential.ErrNotStored)
		})

		t.Run("keeps the values of different hosts apart", func(t *testing.T) {
			isolate(t)
			store := newStore(t, "file")
			_, err := store.Set("api.example.test", "production")
			require.NoError(t, err)
			_, err = store.Set("staging.example.test", "staging")
			require.NoError(t, err)

			production, err := store.Get("api.example.test")
			require.NoError(t, err)
			staging, err := store.Get("staging.example.test")
			require.NoError(t, err)

			require.Equal(t, "production", production.Value)
			require.Equal(t, "staging", staging.Value)
		})
	})

	t.Run("delete", func(t *testing.T) {
		t.Run("removes a value from the OS credential store", func(t *testing.T) {
			isolate(t)
			keyring.MockInit()
			store := newStore(t, "keyring")
			_, err := store.Set("api.example.test", "secret")
			require.NoError(t, err)

			removed, err := store.Delete("api.example.test")

			require.NoError(t, err)
			require.True(t, removed)
			_, err = store.Get("api.example.test")
			require.ErrorIs(t, err, credential.ErrNotStored)
		})

		t.Run("removes a value from the file", func(t *testing.T) {
			isolate(t)
			store := newStore(t, "file")
			_, err := store.Set("api.example.test", "secret")
			require.NoError(t, err)

			removed, err := store.Delete("api.example.test")

			require.NoError(t, err)
			require.True(t, removed)
			_, err = store.Get("api.example.test")
			require.ErrorIs(t, err, credential.ErrNotStored)
		})

		t.Run("reports false when nothing is stored", func(t *testing.T) {
			isolate(t)

			removed, err := newStore(t, "file").Delete("api.example.test")

			require.NoError(t, err)
			require.False(t, removed)
		})
	})

	t.Run("new", func(t *testing.T) {
		t.Run("an unknown GO_CLI_PROJECT_CREDENTIAL_STORE is an error", func(t *testing.T) {
			isolate(t)
			t.Setenv("GO_CLI_PROJECT_CREDENTIAL_STORE", "vault")

			_, err := credential.NewStore()

			require.ErrorContains(t, err, "set it to keyring or file")
		})
	})
}
