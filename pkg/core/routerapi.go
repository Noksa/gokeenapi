package core

import "context"

type Route struct {
    Network   string
    Gateway   string
    Interface string
}

type DeviceInfo struct {
    Model string
    OS    string
    Mode  string
}

type RouterAPI interface {
    DeviceInfo(ctx context.Context) (*DeviceInfo, error)
    AddRoute(ctx context.Context, route Route) error
    DeleteRoute(ctx context.Context, network string) error
    ListRoutes(ctx context.Context) ([]Route, error)
}
