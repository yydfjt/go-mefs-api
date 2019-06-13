package shell

import (
	"bytes"
	"context"
)

type UserPrivMessage struct {
	Address string
	Sk      string
}

type StringList struct {
	ChildLists []string
}

func (fl StringList) String() string {
	var buffer bytes.Buffer
	for i := 0; i < len(fl.ChildLists); i++ {
		buffer.WriteString(fl.ChildLists[i])
		buffer.WriteString("\n")
	}
	return buffer.String()
}

type LfsOpts = func(*RequestBuilder) error

func SetAddress(addr string) LfsOpts {
	return func(rb *RequestBuilder) error {
		rb.Option("address", addr)
		return nil
	}
}

func SetObjectName(objectName string) LfsOpts {
	return func(rb *RequestBuilder) error {
		rb.Option("objectname", objectName)
		return nil
	}
}

func SetPrefixFilter(prefix string) LfsOpts {
	return func(rb *RequestBuilder) error {
		rb.Option("prefix", prefix)
		return nil
	}
}

func SetPolicy(policy int) LfsOpts {
	return func(rb *RequestBuilder) error {
		rb.Option("policy", policy)
		return nil
	}
}

func SetDataCount(dataCount int) LfsOpts {
	return func(rb *RequestBuilder) error {
		rb.Option("datacount", dataCount)
		return nil
	}
}

func SetParityCount(parityCount int) LfsOpts {
	return func(rb *RequestBuilder) error {
		rb.Option("paritycount", parityCount)
		return nil
	}
}

func NeedAvailTime(enabled bool) LfsOpts {
	return func(rb *RequestBuilder) error {
		rb.Option("Avail", enabled)
		return nil
	}
}

func SetSecretKey(sk string) LfsOpts {
	return func(rb *RequestBuilder) error {
		rb.Option("secretekey", sk)
		return nil
	}
}
func SetPassword(pwd string) LfsOpts {
	return func(rb *RequestBuilder) error {
		rb.Option("password", pwd)
		return nil
	}
}

func ForceFlush(enabled bool) LfsOpts {
	return func(rb *RequestBuilder) error {
		rb.Option("force", enabled)
		return nil
	}
}

func UseErasureCodeOrMulRep(enabled bool) LfsOpts {
	return func(rb *RequestBuilder) error {
		rb.Option("policy", enabled)
		return nil
	}
}

func (s *Shell) CreateUser(options ...LfsOpts) (*UserPrivMessage, error) {
	var user UserPrivMessage
	rb := s.Request("create")
	for _, option := range options {
		option(rb)
	}

	if err := rb.Exec(context.Background(), &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Shell) StartUser(address string, options ...LfsOpts) error {
	var res StringList
	rb := s.Request("lfs/start", address)
	for _, option := range options {
		option(rb)
	}
	if err := rb.Exec(context.Background(), &res); err != nil {
		return err
	}
	return nil
}

func (s *Shell) Fsync(options ...LfsOpts) error {
	var res StringList
	rb := s.Request("lfs/fsync")
	for _, option := range options {
		option(rb)
	}

	if err := rb.Exec(context.Background(), &res); err != nil {
		return err
	}
	return nil
}

func (s *Shell) ShowStorage(options ...LfsOpts) error {
	var res string
	rb := s.Request("lfs/show_storage")
	for _, option := range options {
		option(rb)
	}

	if err := rb.Exec(context.Background(), &res); err != nil {
		return err
	}
	return nil
}

func (s *Shell) ShowBalance(options ...LfsOpts) (int64, error) {
	var res int64
	rb := s.Request("lfs/show_balance")
	for _, option := range options {
		option(rb)
	}

	if err := rb.Exec(context.Background(), &res); err != nil {
		return 0, err
	}
	return res, nil
}
