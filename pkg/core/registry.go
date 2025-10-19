package core

import "fmt"

var backends = map[string]func(config map[string]string) (RouterAPI, error){}

func Register(name string, factory func(config map[string]string) (RouterAPI, error)) {
    backends[name] = factory
}

func NewRouter(name string, config map[string]string) (RouterAPI, error) {
    f, ok := backends[name]
    if !ok {
        return nil, fmt.Errorf("unsupported backend: %s", name)
    }
    return f(config)
}
