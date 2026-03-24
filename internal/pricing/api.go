package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	"github.com/aws/aws-sdk-go-v2/service/pricing/types"
)

// FetchFargatePrices ruft On-Demand und Spot-Preise für Fargate aus der AWS Pricing API ab.
// region: die Ziel-Region (eu-central-1 etc.) für den Filter — API-Endpoint ist immer us-east-1.
func FetchFargatePrices(ctx context.Context, region string) (Config, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion("us-east-1"), // Pricing API nur in us-east-1
	)
	if err != nil {
		return Config{}, err
	}
	client := pricing.NewFromConfig(cfg)

	vcpuPrice, err := fetchPrice(ctx, client, region, "Fargate-vCPU-Hours:perCPU")
	if err != nil || vcpuPrice == 0 {
		return Config{}, fmt.Errorf("vCPU-Preis nicht gefunden: %w", err)
	}
	gbPrice, err := fetchPrice(ctx, client, region, "Fargate-GB-Hours:perGB")
	if err != nil || gbPrice == 0 {
		return Config{}, fmt.Errorf("GB-Preis nicht gefunden: %w", err)
	}

	result := Config{
		PriceVCPUHour: vcpuPrice,
		PriceGBHour:   gbPrice,
		Source:        "api",
	}

	// Spot-Preise: Fallback auf ~35% von On-Demand wenn nicht via API verfügbar
	spotVCPU, _ := fetchPrice(ctx, client, region, "Fargate-vCPU-Hours:perCPU-Spot")
	spotGB, _ := fetchPrice(ctx, client, region, "Fargate-GB-Hours:perGB-Spot")
	if spotVCPU > 0 {
		result.SpotPriceVCPUHour = spotVCPU
	} else {
		result.SpotPriceVCPUHour = vcpuPrice * 0.35
	}
	if spotGB > 0 {
		result.SpotPriceGBHour = spotGB
	} else {
		result.SpotPriceGBHour = gbPrice * 0.35
	}

	return result, nil
}

func fetchPrice(ctx context.Context, client *pricing.Client, region, usageType string) (float64, error) {
	out, err := client.GetProducts(ctx, &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonECS"),
		Filters: []types.Filter{
			{Type: types.FilterTypeTermMatch, Field: aws.String("regionCode"), Value: aws.String(region)},
			{Type: types.FilterTypeTermMatch, Field: aws.String("usagetype"), Value: aws.String(usageType)},
		},
		MaxResults: aws.Int32(1),
	})
	if err != nil || len(out.PriceList) == 0 {
		return 0, err
	}
	// Preis aus verschachteltem JSON parsen
	priceStr := extractUSDPrice(out.PriceList[0])
	return strconv.ParseFloat(priceStr, 64)
}

// extractUSDPrice parst den USD-Preis aus dem PriceList-JSON-String.
// Nutzt json.Unmarshal um den Preis zu extrahieren.
func extractUSDPrice(priceJSON string) string {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(priceJSON), &data); err != nil {
		return ""
	}

	// Pfad: terms.OnDemand.*.priceDimensions.*.pricePerUnit.USD
	terms, ok := data["terms"].(map[string]interface{})
	if !ok {
		return ""
	}

	onDemand, ok := terms["OnDemand"].(map[string]interface{})
	if !ok {
		return ""
	}

	// Iteriere über alle OnDemand-Terms
	for _, term := range onDemand {
		termMap, ok := term.(map[string]interface{})
		if !ok {
			continue
		}

		priceDimensions, ok := termMap["priceDimensions"].(map[string]interface{})
		if !ok {
			continue
		}

		// Iteriere über alle priceDimensions
		for _, dim := range priceDimensions {
			dimMap, ok := dim.(map[string]interface{})
			if !ok {
				continue
			}

			pricePerUnit, ok := dimMap["pricePerUnit"].(map[string]interface{})
			if !ok {
				continue
			}

			if usd, ok := pricePerUnit["USD"].(string); ok {
				return usd
			}
		}
	}

	return ""
}
