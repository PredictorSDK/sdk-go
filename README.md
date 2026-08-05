# sdk-go

The official Go client for the [PredictorSDK](https://predictorsdk.com) prediction-market data API.

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

	predictorclient "github.com/PredictorSDK/sdk-go/client"
	"github.com/PredictorSDK/sdk-go/option"
)

func main() {
	client := predictorclient.New(option.WithToken("your-api-key"))

	plans, err := client.GetPlans(context.TODO())
	if err != nil {
		panic(err)
	}
	categories, err := client.GetCategories(context.TODO())
	if err != nil {
		panic(err)
	}
	fmt.Println(plans.Data, categories.Data)
}
```

## Documentation

- [Docs](https://docs.predictorsdk.com)
- [API Reference](https://docs.predictorsdk.com/api-reference)

## License

[MIT](https://github.com/PredictorSDK/sdk-go/blob/main/LICENSE)
