package connectors

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/tommitoan/personal-ops-hub/internal/config"
	"github.com/tommitoan/personal-ops-hub/internal/core"
)

const railwayServicesQuery = `query($projectId:String!){project(id:$projectId){services(first:100){pageInfo{hasNextPage} edges{node{id name serviceInstances(first:20){pageInfo{hasNextPage} edges{node{environmentId activeDeployments{status}}}}}}}}}`

type railwayServiceResponse struct {
	Project *struct {
		Services struct {
			PageInfo struct {
				HasNextPage bool `json:"hasNextPage"`
			} `json:"pageInfo"`
			Edges []struct {
				Node struct {
					ID               string `json:"id"`
					Name             string `json:"name"`
					ServiceInstances struct {
						PageInfo struct {
							HasNextPage bool `json:"hasNextPage"`
						} `json:"pageInfo"`
						Edges []struct {
							Node struct {
								EnvironmentID     string `json:"environmentId"`
								ActiveDeployments []struct {
									Status string `json:"status"`
								} `json:"activeDeployments"`
							} `json:"node"`
						} `json:"edges"`
					} `json:"serviceInstances"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"services"`
	} `json:"project"`
}

// RailwayServices returns deployment-state cards for every service in one project environment.
func (c *Collector) RailwayServices(ctx context.Context, cfg config.Config, now time.Time) []core.Card {
	return c.railwayServices(ctx, cfg, now, "https://backboard.railway.com/graphql/v2")
}

func (c *Collector) railwayServices(ctx context.Context, cfg config.Config, now time.Time, endpoint string) []core.Card {
	link := "https://railway.com/project/" + cfg.RailwayProjectID
	status := card("railway:services", "Railway services", "Railway GraphQL", link, now)
	if cfg.RailwayToken == "" || cfg.RailwayProjectID == "" || cfg.RailwayEnvironmentID == "" {
		status.Summary = "Add RAILWAY_TOKEN, RAILWAY_PROJECT_ID and RAILWAY_ENVIRONMENT_ID"
		return []core.Card{status}
	}
	var data railwayServiceResponse
	if err := c.graphql(ctx, endpoint, cfg.RailwayToken, railwayServicesQuery, map[string]any{"projectId": cfg.RailwayProjectID}, &data); err != nil || data.Project == nil {
		return []core.Card{failed(status)}
	}
	services := data.Project.Services
	if services.PageInfo.HasNextPage || len(services.Edges) == 0 {
		return []core.Card{failed(status)}
	}
	out := make([]core.Card, 0, len(services.Edges)+1)
	active, inactive := 0, 0
	for _, edge := range services.Edges {
		svc := edge.Node
		if svc.ID == "" || svc.Name == "" || svc.ServiceInstances.PageInfo.HasNextPage {
			return []core.Card{failed(status)}
		}
		item := card("railway:service:"+svc.ID, svc.Name, "Railway activeDeployments", link, now)
		item.Window = "selected Railway environment"
		found := false
		for _, instance := range svc.ServiceInstances.Edges {
			if instance.Node.EnvironmentID != cfg.RailwayEnvironmentID {
				continue
			}
			found = true
			if len(instance.Node.ActiveDeployments) == 0 {
				item.Status = "degraded"
				item.Summary = "No active deployment; confirm intended stop/sleep policy"
				inactive++
			} else {
				item.Status = "healthy"
				item.Summary = fmt.Sprintf("%d active deployment(s)", len(instance.Node.ActiveDeployments))
				active++
			}
			break
		}
		if !found {
			item.Status = "degraded"
			item.Summary = "Not present in selected environment"
			inactive++
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	status.Status = "healthy"
	if inactive > 0 {
		status.Status = "degraded"
	}
	status.Summary = fmt.Sprintf("%d services · %d active · %d without active deployment", len(out), active, inactive)
	status.Window = "selected Railway environment"
	return append([]core.Card{status}, out...)
}
