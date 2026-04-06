# sdk-go

The official Go client for the [PredictorSDK](https://predictorsdk.com) matching markets API.

## Installation

```bash
go get github.com/PredictorSDK/sdk-go
```

## Usage

```go
package main

import (
	"context"
	"fmt"

	predictorsdk "github.com/PredictorSDK/sdk-go"
	predictorclient "github.com/PredictorSDK/sdk-go/client"
	"github.com/PredictorSDK/sdk-go/option"
)

func main() {
	client := predictorclient.New(option.WithToken("your-api-key"))

	response, err := client.GetSportsMatchingMarkets(context.TODO(), &predictorsdk.GetSportsMatchingMarketsRequest{
		KalshiEventTicker: []*string{predictorsdk.String("KXMLB-25-NYM-COL-2025-04-03")},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(response.Markets)
}
```

## Documentation

- [Docs](https://docs.predictorsdk.com)
- [API Reference](https://docs.predictorsdk.com/api-reference)

## License

[MIT](https://github.com/PredictorSDK/sdk-go/blob/main/LICENSE)
