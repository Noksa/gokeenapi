package exporter

import (
	"context"
	"encoding/json"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
)

func (*Exporter) interfaceSamples(ctx context.Context, t *target, body json.RawMessage) ([]sample, error) {
	speeds, err := t.rememberInterfaces(body)
	if err != nil {
		return nil, err
	}
	ids, err := selectedInterfaces(body, t.interfaces)
	if err != nil {
		return nil, err
	}
	bodies := []json.RawMessage{body}
	if len(ids) > 0 {
		queries := make([]gokeenrestapi.MonitoringQuery, len(ids))
		for i, id := range ids {
			queries[i] = gokeenrestapi.MonitoringQuery{Path: "/rci/show/interface/stat", Interface: id}
		}
		results := t.session.Read(ctx, queries)
		for _, result := range results {
			if result.Err != nil {
				return nil, result.Err
			}
			bodies = append(bodies, result.Data)
		}
	}
	samples, err := parseSamples("interfaces", bodies, ids)
	return append(samples, speeds...), err
}
