package main

import (
	"context"
	"fmt"

	"github.com/data-preservation-programs/RetrievalBot/pkg/env"
	"github.com/data-preservation-programs/RetrievalBot/pkg/resolver"
)

//nolint:forbidigo,forcetypeassert,exhaustive
func main() {
	// locationResolver := resolver.NewLocationResolver("", 10)
	// x, _ := locationResolver.ResolveIPStr(context.Background(), "6.8.8.6")
	// fmt.Println(os.Getenv("IPINFO_URL"))
	// fmt.Println(x)

	providerResolver, _ := resolver.NewProviderResolver(
		"https://xbj3ebrf2c.execute-api.us-east-2.amazonaws.com/lotus/api.node.glif.io/rpc/v0",
		env.GetString(env.LotusAPIToken, ""),
		10)

	y, _ := providerResolver.ResolveProvider(context.Background(), "f0123")
	fmt.Println(y)

	minerInfo := new(MinerInfo)
	err := p.lotusClient.CallFor(ctx, minerInfo, "Filecoin.StateMinerInfo", provider, nil)
}
