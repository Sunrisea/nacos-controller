package auth

import (
	"context"
	"fmt"

	client2 "github.com/nacos-group/nacos-controller/pkg/nacos/client"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	secretAuthKeyAccessKey = "accessKey"
	secretAuthKeySecretKey = "secretKey"
	secretAuthKeyUsername  = "username"
	secretAuthKeyPassword  = "password"
)

var (
	secretGVK = schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"}
)

type NacosClientParam struct {
	Endpoint   string
	ServerAddr string
	Namespace  string
	AuthInfo   NacosClientAuthInfo
}

type NacosClientAuthInfo struct {
	AccessKey string
	SecretKey string
	Username  string
	Password  string
}

type NacosAuthProvider interface {
	GetNacosClientParams(authRef *v1.ObjectReference, nacosServerParam client2.NacosServerParam, namespace string) (*NacosClientParam, error)
}

type DefaultNacosAuthProvider struct {
	Client client.Client
}

func NewDefaultNacosAuthProvider(c client.Client) NacosAuthProvider {
	return &DefaultNacosAuthProvider{Client: c}
}

func (p *DefaultNacosAuthProvider) GetNacosClientParams(authRef *v1.ObjectReference, nacosServerParam client2.NacosServerParam, namespace string) (*NacosClientParam, error) {
	var authInfo = &NacosClientAuthInfo{}
	if authRef != nil {
		authRef = authRef.DeepCopy()
		authRef.Namespace = namespace
		var err error
		authInfo, err = p.getNacosAuthInfo(authRef)
		if err != nil {
			return nil, err
		}
	}
	if len(nacosServerParam.Endpoint) > 0 {
		return &NacosClientParam{
			Endpoint:  nacosServerParam.Endpoint,
			Namespace: nacosServerParam.Namespace,
			AuthInfo:  *authInfo,
		}, nil
	}
	if len(nacosServerParam.ServerAddr) > 0 {
		return &NacosClientParam{
			ServerAddr: nacosServerParam.ServerAddr,
			Namespace:  nacosServerParam.Namespace,
			AuthInfo:   *authInfo,
		}, nil
	}
	return nil, fmt.Errorf("either endpoint or serverAddr should be set")
}

func (p *DefaultNacosAuthProvider) getNacosAuthInfo(obj *v1.ObjectReference) (*NacosClientAuthInfo, error) {
	switch obj.GroupVersionKind().String() {
	case secretGVK.String():
		return p.getNaocsAuthFromSecret(obj)
	default:
		return nil, fmt.Errorf("unsupported nacos auth reference type: %s", obj.GroupVersionKind().String())
	}
}

func (p *DefaultNacosAuthProvider) getNaocsAuthFromSecret(obj *v1.ObjectReference) (*NacosClientAuthInfo, error) {
	s := v1.Secret{}
	info := NacosClientAuthInfo{}
	if err := p.Client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, &s); err != nil {
		if errors.IsNotFound(err) {
			return &info, nil
		}
		return nil, err
	}
	if v, ok := s.Data[secretAuthKeyAccessKey]; ok && len(v) > 0 {
		info.AccessKey = string(v)
	}
	if v, ok := s.Data[secretAuthKeySecretKey]; ok && len(v) > 0 {
		info.SecretKey = string(v)
	}
	if v, ok := s.Data[secretAuthKeyUsername]; ok && len(v) > 0 {
		info.Username = string(v)
	}
	if v, ok := s.Data[secretAuthKeyPassword]; ok && len(v) > 0 {
		info.Password = string(v)
	}
	return &info, nil
}
