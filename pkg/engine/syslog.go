package engine

import (
	"context"
	"os"

	"github.com/go-errors/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/trufflesecurity/trufflehog/v3/pkg/pb/sourcespb"
	"github.com/trufflesecurity/trufflehog/v3/pkg/sources"
	"github.com/trufflesecurity/trufflehog/v3/pkg/sources/syslog"
)

// ScanSyslog is a source that scans syslog files.
func (e *Engine) ScanSyslog(ctx context.Context, c sources.Config) error {
	connection := &sourcespb.Syslog{
		Protocol:      c.Protocol,
		ListenAddress: c.Address,
		Format:        c.Format,
	}

	if c.CertPath != "" && c.KeyPath != "" {
		cert, err := os.ReadFile(c.CertPath)
		if err != nil {
			return errors.WrapPrefix(err, "could not open TLS cert file", 0)
		}
		connection.TlsCert = string(cert)

		key, err := os.ReadFile(c.KeyPath)
		if err != nil {
			return errors.WrapPrefix(err, "could not open TLS key file", 0)
		}
		connection.TlsKey = string(key)
	}

	var conn anypb.Any
	err := anypb.MarshalFrom(&conn, connection, proto.MarshalOptions{})
	if err != nil {
		return errors.WrapPrefix(err, "error unmarshalling connection", 0)
	}
	source := syslog.Source{}
	err = source.Init(ctx, "trufflehog - syslog", 0, 0, false, &conn, c.Concurrency)
	source.InjectConnection(connection)
	if err != nil {
		logrus.WithError(err).Error("failed to initialize syslog source")
		return err
	}

	e.sourcesWg.Add(1)
	go func() {
		defer e.sourcesWg.Done()
		err := source.Chunks(ctx, e.ChunksChan())
		if err != nil {
			logrus.WithError(err).Fatal("could not scan syslog")
		}
	}()
	return nil
}
