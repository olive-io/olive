package grpc

import "google.golang.org/grpc/status"

func parseErr(err error) error {
	// no error
	switch err {
	case nil:
		return nil
	}

	// grpc error
	s, ok := status.FromError(err)
	if !ok {
		return err
	}

	return s.Err()
}
