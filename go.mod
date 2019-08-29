module github.com/snowdrop/kreate

require (
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20190711103511-473e67f1d7d2 // indirect
	github.com/elazarl/goproxy/ext v0.0.0-20190711103511-473e67f1d7d2 // indirect
	github.com/fatih/color v1.7.0
	github.com/ghodss/yaml v1.0.0
	github.com/gobwas/glob v0.2.3
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/mattn/go-colorable v0.1.1
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.1
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.4.0
	golang.org/x/crypto v0.0.0-20190611184440-5c40567a22f8
	gopkg.in/AlecAivazis/survey.v1 v1.8.4
	halkyon.io/api v1.0.0-beta.4
	k8s.io/api v0.0.0-20190725062911-6607c48751ae
	k8s.io/apimachinery v0.0.0-20190719140911-bfcf53abc9f8
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/code-generator v0.0.0-20190828114452-79369ab2ae19 // indirect
)

replace (
	k8s.io/api => k8s.io/api v0.0.0-20190516230258-a675ac48af67
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190516231937-17bc0b7fcef5
	k8s.io/client-go => k8s.io/client-go v11.0.1-0.20190516230509-ae8359b20417+incompatible
)
