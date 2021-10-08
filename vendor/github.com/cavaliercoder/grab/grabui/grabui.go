package grabui

import (
	"context"

	"github.com/cavaliercoder/grab"
)

func GetBatch(
	ctx context.Context,
	workers int,
	dst string,
	urlStrs ...string,
) (<-chan *grab.Response, error) {
	reqs := make([]*grab.Request, len(urlStrs))
	for i := 0; i < len(urlStrs); i++ {
		req, err := grab.NewRequest(dst, urlStrs[i])
		if err != nil {
			return nil, err
		}
		req = req.WithContext(ctx)
		reqs[i] = req
	}

	ui := NewConsoleClient(grab.DefaultClient)
	return ui.Do(ctx, workers, reqs...), nil
}
