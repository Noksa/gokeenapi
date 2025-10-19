
package keenetic

import (
    "context"
    "fmt"
    "github.com/kam1k88/gokeenapi/pkg/core"
)

type Client struct {
    Host  string
    Token string
}

func New(config map[string]string) (core.RouterAPI, error) {
    return &Client{
        Host:  config["host"],
        Token: config["token"],
    }, nil
}

func init() {
    core.Register("keenetic", New)
}

func (c *Client) DeviceInfo(ctx context.Context) (*core.DeviceInfo, error) {
    // Заглушка — позже вызов реального Keenetic API
    return &core.DeviceInfo{Model: "Keenetic (mock)", OS: "4.x", Mode: "router"}, nil
}

func (c *Client) AddRoute(ctx context.Context, route core.Route) error {
    fmt.Printf("[Keenetic] Добавление маршрута %s -> %s\n", route.Network, route.Gateway)
    return nil
}

func (c *Client) DeleteRoute(ctx context.Context, network string) error {
    fmt.Printf("[Keenetic] Удаление маршрута %s\n", network)
    return nil
}

func (c *Client) ListRoutes(ctx context.Context) ([]core.Route, error) {
    return []core.Route{{Network: "10.0.0.0/24", Gateway: "192.168.1.1", Interface: "ISP"}}, nil
}
