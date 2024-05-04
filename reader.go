package overturemaps

import (
	"fmt"
	"io"

	"github.com/parquet-go/parquet-go"
)

func ReadFile[T any](r io.ReaderAt) ([]T, error) {
	reader := parquet.NewGenericReader[T](r)
	fmt.Println(reader.Schema())

	rows := make([]T, 0, reader.NumRows())
	for i := 0; int64(i) < reader.NumRows(); i++ {
		row := make([]T, 1)
		if _, err := reader.Read(row); err != nil {
			return nil, err
		}
		rows = append(rows, row[0])
	}

	return rows, nil
}
