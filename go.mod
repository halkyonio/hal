module halkyon.io/hal

require (
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20190711103511-473e67f1d7d2 // indirect
	github.com/elazarl/goproxy/ext v0.0.0-20190711103511-473e67f1d7d2 // indirect
	github.com/fatih/color v1.7.0
	github.com/frankban/quicktest v1.5.0 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/mattn/go-colorable v0.1.1
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b
	github.com/mholt/archiver v0.0.0-20190623220050-33320f6f7306
	github.com/pierrec/lz4 v2.3.0+incompatible // indirect
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20191107222254-f4817d981bb6
	gopkg.in/AlecAivazis/survey.v1 v1.8.4
	halkyon.io/api v1.0.0-rc.5.0.20200228144735-fce240b08314
	k8s.io/api v0.0.0-20190918195907-bd6ac527cfd2
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v11.0.1-0.20190805182715-88a2adca7e76+incompatible
	k8s.io/kubectl v0.0.0-20190831163037-3b58a944563f
)

replace (
	k8s.io/api => k8s.io/api v0.0.0-20190516230258-a675ac48af67
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go => k8s.io/client-go v11.0.1-0.20190516230509-ae8359b20417+incompatible
)

go 1.13
