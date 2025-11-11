package cmd

// Command names
const (
	CmdAddRoutes        = "add-routes"
	CmdDeleteRoutes     = "delete-routes"
	CmdShowInterfaces   = "show-interfaces"
	CmdAddAwg           = "add-awg"
	CmdUpdateAwg        = "update-awg"
	CmdAddDnsRecords    = "add-dns-records"
	CmdDeleteDnsRecords = "delete-dns-records"
	CmdDeleteKnownHosts = "delete-known-hosts"
	CmdExec             = "exec"
	CmdScheduler        = "scheduler"
	CmdVersion          = "version"
)

// Built-in commands that should skip initialization
const (
	CmdCompletion = "completion"
	CmdHelp       = "help"
)

// Command aliases
var (
	AliasesAddRoutes        = []string{"addroutes", "ar"}
	AliasesDeleteRoutes     = []string{"deleteroutes", "dr"}
	AliasesShowInterfaces   = []string{"showinterfaces", "si", "showinterface", "show-interface"}
	AliasesAddAwg           = []string{"addawg", "aawg"}
	AliasesUpdateAwg        = []string{"updateawg", "uawg"}
	AliasesAddDnsRecords    = []string{"adddnsrecords", "adr"}
	AliasesDeleteDnsRecords = []string{"deletednsrecords", "ddr"}
	AliasesDeleteKnownHosts = []string{"deleteknownhosts", "dkh"}
	AliasesExec             = []string{"e"}
	AliasesScheduler        = []string{"schedule", "sched"}
)
