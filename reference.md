# Reference
<details><summary><code>client.GetSportsMatchingMarkets() -> *sdkgo.SportsMatchingResponse</code></summary>
<dl>
<dd>

#### 📝 Description

<dl>
<dd>

<dl>
<dd>

Find cross-platform market matches for sports events. Provide a Kalshi event ticker or Polymarket market slug to look up the equivalent market on other platforms. When called without parameters, returns all currently matched sports markets.
</dd>
</dl>
</dd>
</dl>

#### 🔌 Usage

<dl>
<dd>

<dl>
<dd>

```go
request := &sdkgo.GetSportsMatchingMarketsRequest{}
client.GetSportsMatchingMarkets(
        context.TODO(),
        request,
    )
}
```
</dd>
</dl>
</dd>
</dl>

#### ⚙️ Parameters

<dl>
<dd>

<dl>
<dd>

**kalshiEventTicker:** `*string` — Kalshi event ticker(s) to find matching markets for (e.g. `KXNFLGAME-25AUG16ARIDEN`). Provide the parameter multiple times for multiple tickers. Only one filter type may be used per request.
    
</dd>
</dl>

<dl>
<dd>

**polymarketMarketSlug:** `*string` — Polymarket market slug(s) to find matching markets for (e.g. `nfl-ari-den-2025-08-16`). Provide the parameter multiple times for multiple slugs. Only one filter type may be used per request.
    
</dd>
</dl>

<dl>
<dd>

**predictMarketID:** `*string` — Predict market ID(s) to find matching markets for (e.g. `110629`). Provide the parameter multiple times for multiple IDs. Only one filter type may be used per request.
    
</dd>
</dl>

<dl>
<dd>

**sxbetMarketID:** `*string` — SX Bet market ID(s) to find matching markets for (e.g. `0x4c000abdbf197ef32ecdf15561b1d636f1e5b02629f466678757fd83e2ec3599`). Provide the parameter multiple times for multiple IDs. Only one filter type may be used per request.
    
</dd>
</dl>
</dd>
</dl>


</dd>
</dl>
</details>

