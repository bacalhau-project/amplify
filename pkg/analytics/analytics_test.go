package analytics

import (
	"context"
	"os"
	"testing"

	"github.com/bacalhau-project/amplify/pkg/db"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gotest.tools/assert"
)

func Test_analyticsRepository_ParseAndStore(t *testing.T) {
	log.Logger = zerolog.New(zerolog.ConsoleWriter{
		Out:     os.Stderr,
		NoColor: true,
		PartsExclude: []string{
			zerolog.TimestampFieldName,
		},
	})
	zerolog.SetGlobalLevel(zerolog.TraceLevel)

	type args struct {
		ctx    context.Context
		id     uuid.UUID
		result string
	}
	tests := []struct {
		name           string
		args           args
		queryKey       string
		typeKey        string
		expectedResult int
		wantErr        bool
	}{
		{
			"simple",
			args{ctx: context.Background(), id: uuid.New(), result: `{"content-type": "image/png"}`},
			"content-type",
			"image/png",
			1,
			false,
		},
		{
			"multiline",
			args{ctx: context.Background(), id: uuid.New(), result: `{"content-type": "image/png"}
			{"content-type": "image/png"}`},
			"content-type",
			"image/png",
			2,
			false,
		},
		{
			"sameline",
			args{ctx: context.Background(), id: uuid.New(), result: `{"content-type": "image/png"} {"content-type": "image/png"}`},
			"content-type",
			"image/png",
			2,
			false,
		},
		{
			"big",
			args{ctx: context.Background(), id: uuid.New(), result: `{"Content-Type":"image/jpeg"} {"Content-Type":"image/jpeg"} {"Content-Type":"image/jpeg"} {"Content-Type":"image/jpeg"} {"Content-Type":"image/jpeg"} {"Content-Type":"image/jpeg"} {"Content-Type":"image/jpeg"} {"Content-Type":"image/jpeg"} {"Content-Type":"image/jpeg"} {"Content-Type":"image/jpeg"} {"Content-Type":"image/jpeg"} {"Content-Type":"image/jpeg"} {"Content-Type":"image/jpeg"} {"Content-Type":"image/jpeg"} {"Content-Type":"image/svg+xml"} {"Content-Type":"image/jpeg"} {"Content-Type":"image/svg+xml"} {"Content-Type":"image/svg+xml"}`},
			"content-type",
			"image/svg+xml",
			3,
			false,
		},
		{
			"random_crap",
			args{ctx: context.Background(), id: uuid.New(), result: `sing subdir: /25\nprocessing input_file: /inputs/image\nprocessing input_file: /inputs/image/big\nprocessing input_file: /inputs/image/big/image1.jpg\n-e :\\n\\tdir  = \"\"\\n\\tbase = \"image1\"\\n\\text  = \"jpg\"\ninput_file: /inputs/image/big/image1.jpg\n`},
			"",
			"",
			0,
			false,
		},
		{
			"random_crap_with_json_in_middle",
			args{ctx: context.Background(), id: uuid.New(), result: `using subdir: /25\nprocessing input_file: /inputs/image\nprocessing 
			{"Content-Type":"image/jpeg"}
			input_file: /inputs/image/big\nprocessing input_file: /inputs/image/big/image1.jpg\n-e :\\n\\tdir  = \"\"\\n\\tbase = \"image1\"\\n\\text  = \"jpg\"\ninput_file: /inputs/image/big/image1.jpg\n`},
			"Content-Type",
			"image/jpeg",
			1,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &analyticsRepository{
				database: db.NewInMemDB(),
			}
			if err := r.ParseAndStore(tt.args.ctx, tt.args.id, tt.args.result); (err != nil) != tt.wantErr {
				t.Errorf("analyticsRepository.ParseAndStore() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.expectedResult > 0 {
				res, err := r.QueryTopResultsByKey(tt.args.ctx, QueryTopResultsByKeyParams{Key: tt.queryKey, PageSize: 1, Sort: "-count"})
				assert.NilError(t, err)
				for _, v := range res.Results {
					if v.Key == tt.typeKey {
						assert.Equal(t, v.Value, int64(tt.expectedResult))
						break
					}
				}
			}
		})
	}
}
