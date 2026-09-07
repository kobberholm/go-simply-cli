package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	sdk "github.com/kobberholm/go-simply-sdk"
)

type Metadata struct {
	RateLimit          string `json:"rate_limit,omitempty"`
	RateLimitRemaining string `json:"rate_limit_remaining,omitempty"`
}

type ProductsResult struct {
	Data []sdk.Product `json:"data"`
	Metadata
}

type RecordsResult struct {
	Data []sdk.Record `json:"data"`
	Metadata
}

type ZoneResult struct {
	Data sdk.Zone `json:"data"`
	Metadata
}

func Products(writer io.Writer, format string, products []sdk.Product, response sdk.Response) error {
	sort.SliceStable(products, func(left, right int) bool {
		return productKey(products[left]) < productKey(products[right])
	})
	if format == "json" {
		return json.NewEncoder(writer).Encode(ProductsResult{Data: products, Metadata: metadata(response)})
	}
	w := tabwriter.NewWriter(writer, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "OBJECT\tTYPE\tNAME\tSTATUS"); err != nil {
		return err
	}
	for _, product := range products {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", product.Object, product.Type, product.Name, product.Status); err != nil {
			return err
		}
	}
	if len(products) == 0 {
		if _, err := fmt.Fprintln(w, "No products found.\t\t\t"); err != nil {
			return err
		}
	}
	return w.Flush()
}

func Records(writer io.Writer, format string, records []sdk.Record, response sdk.Response) error {
	sort.SliceStable(records, func(left, right int) bool {
		return recordKey(records[left]) < recordKey(records[right])
	})
	if format == "json" {
		return json.NewEncoder(writer).Encode(RecordsResult{Data: records, Metadata: metadata(response)})
	}
	w := tabwriter.NewWriter(writer, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "ID\tNAME\tTYPE\tVALUE\tTTL"); err != nil {
		return err
	}
	for _, record := range records {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", record.ID, record.Name, record.Type, record.Value, ttl(record.TTL)); err != nil {
			return err
		}
	}
	if len(records) == 0 {
		if _, err := fmt.Fprintln(w, "No DNS records found.\t\t\t\t"); err != nil {
			return err
		}
	}
	return w.Flush()
}

func Zone(writer io.Writer, format string, zone sdk.Zone, response sdk.Response) error {
	sort.SliceStable(zone.Records, func(left, right int) bool {
		return recordKey(zone.Records[left]) < recordKey(zone.Records[right])
	})
	if format == "json" {
		return json.NewEncoder(writer).Encode(ZoneResult{Data: zone, Metadata: metadata(response)})
	}
	if _, err := fmt.Fprintf(writer, "Zone: %s\n", zone.Name); err != nil {
		return err
	}
	return Records(writer, "table", zone.Records, response)
}

func metadata(response sdk.Response) Metadata {
	return Metadata{RateLimit: response.RateLimitLimit, RateLimitRemaining: response.RateLimitRemaining}
}

func productKey(product sdk.Product) string {
	return product.Object + "\x00" + product.Type + "\x00" + product.Name
}

func recordKey(record sdk.Record) string {
	return record.Name + "\x00" + record.Type + "\x00" + record.Value + "\x00" + record.ID
}

func ttl(value *int) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(*value)
}
