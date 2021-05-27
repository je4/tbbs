module github.com/je4/tbbs/v2

go 1.16

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/blend/go-sdk v1.20210413.2
	github.com/dgraph-io/badger v1.6.2
	github.com/dgraph-io/ristretto v0.0.3 // indirect
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/goph/emperror v0.17.2
	github.com/je4/bagarc/v2 v2.0.2
	github.com/je4/sshtunnel/v2 v2.0.0-20210324104725-ab38247e5ffa
	github.com/machinebox/progress v0.2.0
	github.com/matryer/is v1.4.0 // indirect
	github.com/oliveagle/jsonpath v0.0.0-20180606110733-2e52cf6e6852 // indirect
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/pinpt/go-common v9.1.81+incompatible
	github.com/pkg/sftp v1.13.0
	github.com/spf13/pflag v1.0.5
	github.com/tidwall/transform v0.0.0-20201103190739-32f242e2dbde
	golang.org/x/crypto v0.0.0-20210421170649-83a5a9bb288b
	golang.org/x/net v0.0.0-20210423184538-5f58ad60dda6 // indirect
	golang.org/x/sys v0.0.0-20210426230700-d19ff857e887 // indirect
	golang.org/x/text v0.3.6
)

replace github.com/je4/tbbs/v2 => ../tbbs
