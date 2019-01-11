# go-badge [![GoDoc](https://godoc.org/github.com/narqo/go-badge?status.svg)](https://godoc.org/github.com/narqo/go-badge)

go-badge is a library to render shield badges to SVG.

## Installation

Using `go get`

```
go get github.com/narqo/go-badge
```

## Usage

```go
import (
	"os"

	"github.com/narqo/go-badge"
)

func main() {
	if err := badge.Render("godoc", "reference", "#5272B4", os.Stdout); err != nil {
		panic(err)
	}
}
```

Hope `example/` directory will have more examples in future.

## Contribution and Feedback

Contributing is more than welcome. Create an issue if you ses any problem in the code or send a PR with fixes if you'd like.

## License

MIT

---

All the kudos should go to the great [Shields.io](https://github.com/badges/shields) specification project.
