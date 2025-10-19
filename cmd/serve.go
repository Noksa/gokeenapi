package cmd

import (
    "context"
    "fmt"
    "github.com/spf13/cobra"
    "github.com/kam1k88/gokeenapi/pkg/core"
    _ "github.com/kam1k88/gokeenapi/pkg/backends/keenetic" // импортирует init()
)

func NewRootCmd() *cobra.Command {
    var backend, host, token string
    cmd := &cobra.Command{
        Use:   "gokeenapi",
        Short: "Универсальный инструмент управления роутерами",
        RunE: func(cmd *cobra.Command, args []string) error {
            cfg := map[string]string{"host": host, "token": token}
            router, err := core.NewRouter(backend, cfg)
            if err != nil {
                return err
            }
            info, _ := router.DeviceInfo(context.Background())
            fmt.Printf("✅ Подключено: %s (%s)\n", info.Model, info.OS)
            routes, _ := router.ListRoutes(context.Background())
            fmt.Println("Маршруты:", routes)
            return nil
        },
    }

    cmd.Flags().StringVar(&backend, "backend", "keenetic", "Тип роутера")
    cmd.Flags().StringVar(&host, "host", "http://192.168.1.1", "Адрес роутера")
    cmd.Flags().StringVar(&token, "token", "", "Токен авторизации")

    return cmd
}
