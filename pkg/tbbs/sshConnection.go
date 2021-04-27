package tbbs

import (
	"crypto/sha512"
	"fmt"
	"github.com/goph/emperror"
	"github.com/op/go-logging"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
)

type SSHConnection struct {
	client  *ssh.Client
	config  *ssh.ClientConfig
	address string
	log     *logging.Logger
}

func NewSSHConnection(address, user string, config *ssh.ClientConfig, log *logging.Logger) (*SSHConnection, error) {
	// create copy of config with user
	newConfig := &ssh.ClientConfig{
		Config:            config.Config,
		User:              user,
		Auth:              config.Auth,
		HostKeyCallback:   config.HostKeyCallback,
		BannerCallback:    config.BannerCallback,
		ClientVersion:     config.ClientVersion,
		HostKeyAlgorithms: config.HostKeyAlgorithms,
		Timeout:           config.Timeout,
	}

	sc := &SSHConnection{
		client:  nil,
		log:     log,
		config:  newConfig,
		address: address,
	}
	// connect
	if err := sc.Connect(); err != nil {
		return nil, emperror.Wrapf(err, "cannot connect to %s@%s", user, address)
	}
	return sc, nil
}

func (sc *SSHConnection) Connect() error {
	var err error
	sc.client, err = ssh.Dial("tcp", sc.address, sc.config)
	if err != nil {
		return emperror.Wrapf(err, "unable to connect to %v", sc.address)
	}

	return nil
}

func (sc *SSHConnection) Close() {
	sc.client.Close()
}

func (sc *SSHConnection) GetSFTPClient() (*sftp.Client, error) {
	sftpclient, err := sftp.NewClient(sc.client)
	if err != nil {
		sc.log.Infof("cannot get sftp subsystem - reconnecting to %s@%s", sc.client.User(), sc.address)
		if err := sc.Connect(); err != nil {
			return nil, emperror.Wrapf(err, "cannot connect with ssh to %s@%s", sc.client.User(), sc.address)
		}
		sftpclient, err = sftp.NewClient(sc.client)
		if err != nil {
			return nil, emperror.Wrapf(err, "cannot create sftp client on %s@%s", sc.client.User(), sc.address)
		}
	}
	return sftpclient, nil
}

func (sc *SSHConnection) ReadFile(path string, w io.Writer) (int64, error) {
	sftpclient, err := sc.GetSFTPClient()
	if err != nil {
		return 0, emperror.Wrap(err, "unable to create SFTP session")
	}
	defer sftpclient.Close()

	r, err := sftpclient.Open(path)
	if err != nil {
		return 0, emperror.Wrapf(err, "cannot open remote file %s", path)
	}
	defer r.Close()
	written, err := io.Copy(w, r)
	if err != nil {
		return 0, emperror.Wrap(err, "cannot copy data")
	}
	return written, nil
}

func (sc *SSHConnection) WriteFile(path string, r io.Reader) (int64, string, error) {
	sftpclient, err := sc.GetSFTPClient()
	if err != nil {
		return 0, "", emperror.Wrap(err, "unable to create SFTP session")
	}
	defer sftpclient.Close()

	w, err := sftpclient.Create(path)
	if err != nil {
		return 0, "", emperror.Wrapf(err, "cannot create remote file %s", path)
	}
	shaSink := sha512.New()
	dest := io.MultiWriter(w, shaSink)
	written, err := io.Copy(dest, r)
	if err != nil {
		return 0, "", emperror.Wrap(err, "cannot copy data")
	}
	return written, fmt.Sprintf("%x", shaSink.Sum(nil)), nil
}
