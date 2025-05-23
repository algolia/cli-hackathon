package importentries

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/algolia/cli/pkg/httpmock"
	"github.com/algolia/cli/test"
)

func Test_runImportCmd(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "entries.json")
	err := os.WriteFile(
		tmpFile,
		[]byte(
			`{"language":"en","word":"test","objectID":"test"}`,
		),
		0o600,
	)
	require.NoError(t, err)

	tests := []struct {
		name    string
		cli     string
		stdin   string
		wantOut string
		wantErr string
	}{
		{
			name:    "from stdin",
			cli:     "stopwords -F -",
			stdin:   `{"language":"en","word":"test","objectID":"test"}`,
			wantOut: "✓ Successfully imported 1 entries on stopwords in",
		},
		{
			name:    "from file",
			cli:     fmt.Sprintf("stopwords -F '%s'", tmpFile),
			wantOut: "✓ Successfully imported 1 entries on stopwords in",
		},
		{
			name:    "from stdin with invalid JSON",
			cli:     "stopwords -F -",
			stdin:   `{"language":"en","word":"test"}`,
			wantErr: "X Found 1 error (out of 1 entries) while parsing the file:\n  line 1: objectID is missing\n",
		},
		{
			name: "from stdin with invalid JSON (multiple operations)",
			cli:  "stopwords -F -",
			stdin: `{"word":"test","objectID":"test"},
			{"language":"fr","objectID":"testFr"}`,
			wantErr: "X Found 2 errors (out of 2 entries) while parsing the file:\n  line 1: invalid character ',' after top-level value\n  line 2: word is missing\n",
		},
		{
			name:    "from stdin with invalid JSON (1 entry) with --continue-on-error",
			cli:     "stopwords -F - --continue-on-error",
			stdin:   `{"language":"en"}`,
			wantErr: "X Found 1 error (out of 1 entries) while parsing the file:\n  line 1: objectID is missing\n",
		},
		{
			name: "from stdin with invalid JSON (2 entries) with --continue-on-error",
			cli:  "stopwords -F - --continue-on-error",
			stdin: `{"language":"en","objectID":"test"}
			{"language":"en","word":"test"}`,
			wantErr: "X Found 2 errors (out of 2 entries) while parsing the file:\n  line 1: word is missing\n  line 2: objectID is missing\n",
		},
		{
			name:    "missing file flag",
			cli:     "stopwords",
			wantErr: "required flag(s) \"file\" not set",
		},
		{
			name:    "non-existant file",
			cli:     "stopwords -F /tmp/foo",
			wantErr: "open /tmp/foo: no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httpmock.Registry{}
			if tt.wantErr == "" {
				r.Register(
					httpmock.REST("POST", "1/dictionaries/stopwords/batch"),
					httpmock.JSONResponse(search.UpdatedAtResponse{}),
				)
			}
			defer r.Verify(t)

			f, out := test.NewFactory(true, &r, nil, tt.stdin)
			cmd := NewImportCmd(f, nil)
			out, err := test.Execute(cmd, tt.cli, out)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			assert.Contains(t, out.String(), tt.wantOut)
		})
	}
}
