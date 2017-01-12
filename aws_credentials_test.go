package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/stretchr/testify/assert"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/watch"
)

type fakeKubeClient struct {
	secrets         map[string]*fakeSecrets
	namespaces      *fakeNamespaces
	serviceaccounts map[string]*fakeServiceAccounts
}

type fakeSecrets struct {
	store map[string]*api.Secret
}

type fakeServiceAccounts struct {
	store map[string]*api.ServiceAccount
}

type fakeNamespaces struct {
	store map[string]api.Namespace
}

func (f *fakeKubeClient) Secrets(namespace string) unversioned.SecretsInterface {
	return f.secrets[namespace]
}

func (f *fakeKubeClient) Namespaces() unversioned.NamespaceInterface {
	return f.namespaces
}

func (f *fakeKubeClient) ServiceAccounts(namespace string) unversioned.ServiceAccountsInterface {
	return f.serviceaccounts[namespace]
}

func (f *fakeSecrets) Create(secret *api.Secret) (*api.Secret, error) {
	_, ok := f.store[secret.Name]

	if ok {
		return nil, fmt.Errorf("Secret %v already exists", secret.Name)
	}

	f.store[secret.Name] = secret
	return secret, nil
}

func (f *fakeSecrets) Update(secret *api.Secret) (*api.Secret, error) {
	_, ok := f.store[secret.Name]

	if !ok {
		return nil, fmt.Errorf("Secret: %v not found", secret.Name)
	}

	f.store[secret.Name] = secret
	return secret, nil
}

func (f *fakeSecrets) Get(name string) (*api.Secret, error) {
	secret, ok := f.store[name]

	if !ok {
		return nil, fmt.Errorf("Secret with name: %v not found", name)
	}

	return secret, nil
}

func (f *fakeSecrets) Delete(name string) error                            { return nil }
func (f *fakeSecrets) List(opts api.ListOptions) (*api.SecretList, error)  { return nil, nil }
func (f *fakeSecrets) Watch(opts api.ListOptions) (watch.Interface, error) { return nil, nil }

func (f *fakeServiceAccounts) Get(name string) (*api.ServiceAccount, error) {
	serviceAccount, ok := f.store[name]

	if !ok {
		return nil, fmt.Errorf("Failed to find service account: %v", name)
	}

	return serviceAccount, nil
}

func (f *fakeServiceAccounts) Update(serviceAccount *api.ServiceAccount) (*api.ServiceAccount, error) {
	serviceAccount, ok := f.store[serviceAccount.Name]

	if !ok {
		return nil, fmt.Errorf("Service account: %v not found", serviceAccount.Name)
	}

	f.store[serviceAccount.Name] = serviceAccount
	return serviceAccount, nil
}

func (f *fakeServiceAccounts) Delete(name string) error {
	_, ok := f.store[name]

	if !ok {
		return fmt.Errorf("Service account: %v not found", name)
	}

	delete(f.store, name)
	return nil
}

func (f *fakeServiceAccounts) Create(serviceAccount *api.ServiceAccount) (*api.ServiceAccount, error) {
	return nil, nil
}
func (f *fakeServiceAccounts) List(opts api.ListOptions) (*api.ServiceAccountList, error) {
	return nil, nil
}
func (f *fakeServiceAccounts) Watch(opts api.ListOptions) (watch.Interface, error) { return nil, nil }

func (f *fakeNamespaces) List(opts api.ListOptions) (*api.NamespaceList, error) {
	namespaces := []api.Namespace{}

	for _, v := range f.store {
		namespaces = append(namespaces, v)
	}

	return &api.NamespaceList{Items: namespaces}, nil
}

func (f *fakeNamespaces) Create(item *api.Namespace) (*api.Namespace, error)   { return nil, nil }
func (f *fakeNamespaces) Get(name string) (result *api.Namespace, err error)   { return nil, nil }
func (f *fakeNamespaces) Delete(name string) error                             { return nil }
func (f *fakeNamespaces) Update(item *api.Namespace) (*api.Namespace, error)   { return nil, nil }
func (f *fakeNamespaces) Watch(opts api.ListOptions) (watch.Interface, error)  { return nil, nil }
func (f *fakeNamespaces) Finalize(item *api.Namespace) (*api.Namespace, error) { return nil, nil }
func (f *fakeNamespaces) Status(item *api.Namespace) (*api.Namespace, error)   { return nil, nil }

type fakeEcrClient struct{}

func (f *fakeEcrClient) GetAuthorizationToken(input *ecr.GetAuthorizationTokenInput) (*ecr.GetAuthorizationTokenOutput, error) {
	return &ecr.GetAuthorizationTokenOutput{
		AuthorizationData: []*ecr.AuthorizationData{
			&ecr.AuthorizationData{
				AuthorizationToken: aws.String("fakeToken"),
				ProxyEndpoint:      aws.String("fakeEndpoint"),
			},
		},
	}, nil
}

func newFakeKubeClient() *fakeKubeClient {
	return &fakeKubeClient{
		secrets: map[string]*fakeSecrets{
			"namespace1": &fakeSecrets{
				store: map[string]*api.Secret{},
			},
			"namespace2": &fakeSecrets{
				store: map[string]*api.Secret{},
			},
			"kube-system": &fakeSecrets{
				store: map[string]*api.Secret{},
			},
		},
		namespaces: &fakeNamespaces{store: map[string]api.Namespace{
			"namespace1": api.Namespace{
				ObjectMeta: api.ObjectMeta{
					Name: "namespace1",
				},
			},
			"namespace2": api.Namespace{
				ObjectMeta: api.ObjectMeta{
					Name: "namespace2",
				},
			},
			"kube-system": api.Namespace{
				ObjectMeta: api.ObjectMeta{
					Name: "kube-system",
				},
			},
		}},
		serviceaccounts: map[string]*fakeServiceAccounts{
			"namespace1": &fakeServiceAccounts{
				store: map[string]*api.ServiceAccount{
					"default": &api.ServiceAccount{
						ObjectMeta: api.ObjectMeta{
							Name: "default",
						},
					},
				},
			},
			"namespace2": &fakeServiceAccounts{
				store: map[string]*api.ServiceAccount{
					"default": &api.ServiceAccount{
						ObjectMeta: api.ObjectMeta{
							Name: "default",
						},
					},
				},
			},
			"kube-system": &fakeServiceAccounts{
				store: map[string]*api.ServiceAccount{
					"default": &api.ServiceAccount{
						ObjectMeta: api.ObjectMeta{
							Name: "default",
						},
					},
				},
			},
		},
	}
}

func newFakeEcrClient() *fakeEcrClient {
	return &fakeEcrClient{}
}

func TestgetECRAuthorizationKey(t *testing.T) {
	kubeClient := newFakeKubeClient()
	ecrClient := newFakeEcrClient()

	c := &controller{kubeClient, ecrClient}

	token, err := c.getECRAuthorizationKey()

	assert.Equal(t, "fakeToken", *token.AuthorizationToken)
	assert.Equal(t, "fakeEndpoint", *token.ProxyEndpoint)
	assert.Nil(t, err)
}

func TestProcessOnce(t *testing.T) {
	kubeClient := newFakeKubeClient()
	ecrClient := newFakeEcrClient()

	c := &controller{kubeClient, ecrClient}
	err := c.process()
	assert.Nil(t, err)

	secret, err := c.kubeClient.Secrets("namespace1").Get(*argDefaultSecretName)
	assert.Nil(t, err)
	assert.Equal(t, *argDefaultSecretName, secret.Name)
	assert.Equal(t, map[string][]byte{
		".dockerconfigjson": []byte(fmt.Sprintf(dockerJSONTemplate, "fakeEndpoint", "fakeToken")),
	}, secret.Data)
	assert.Equal(t, api.SecretType("kubernetes.io/dockerconfigjson"), secret.Type)

	secret, err = c.kubeClient.Secrets("namespace2").Get(*argDefaultSecretName)
	assert.Nil(t, err)
	assert.Equal(t, *argDefaultSecretName, secret.Name)
	assert.Equal(t, map[string][]byte{
		".dockerconfigjson": []byte(fmt.Sprintf(dockerJSONTemplate, "fakeEndpoint", "fakeToken")),
	}, secret.Data)
	assert.Equal(t, api.SecretType("kubernetes.io/dockerconfigjson"), secret.Type)

	_, err = c.kubeClient.Secrets("kube-system").Get(*argDefaultSecretName)
	assert.NotNil(t, err)

	serviceAccount, err := c.kubeClient.ServiceAccounts("namespace1").Get("default")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(serviceAccount.ImagePullSecrets))
	assert.Equal(t, *argDefaultSecretName, serviceAccount.ImagePullSecrets[0].Name)

	serviceAccount, err = c.kubeClient.ServiceAccounts("namespace2").Get("default")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(serviceAccount.ImagePullSecrets))
	assert.Equal(t, *argDefaultSecretName, serviceAccount.ImagePullSecrets[0].Name)
}

func TestProcessTwice(t *testing.T) {
	kubeClient := newFakeKubeClient()
	ecrClient := newFakeEcrClient()

	c := &controller{kubeClient, ecrClient}
	err := c.process()
	assert.Nil(t, err)
	// test processing twice for idempotency
	err = c.process()
	assert.Nil(t, err)

	secret, err := c.kubeClient.Secrets("namespace1").Get(*argDefaultSecretName)
	assert.Nil(t, err)
	assert.Equal(t, *argDefaultSecretName, secret.Name)
	assert.Equal(t, map[string][]byte{
		".dockerconfigjson": []byte(fmt.Sprintf(dockerJSONTemplate, "fakeEndpoint", "fakeToken")),
	}, secret.Data)
	assert.Equal(t, api.SecretType("kubernetes.io/dockerconfigjson"), secret.Type)

	secret, err = c.kubeClient.Secrets("namespace2").Get(*argDefaultSecretName)
	assert.Nil(t, err)
	assert.Equal(t, *argDefaultSecretName, secret.Name)
	assert.Equal(t, map[string][]byte{
		".dockerconfigjson": []byte(fmt.Sprintf(dockerJSONTemplate, "fakeEndpoint", "fakeToken")),
	}, secret.Data)
	assert.Equal(t, api.SecretType("kubernetes.io/dockerconfigjson"), secret.Type)

	_, err = c.kubeClient.Secrets("kube-system").Get(*argDefaultSecretName)
	assert.NotNil(t, err)

	serviceAccount, err := c.kubeClient.ServiceAccounts("namespace1").Get("default")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(serviceAccount.ImagePullSecrets))
	assert.Equal(t, *argDefaultSecretName, serviceAccount.ImagePullSecrets[0].Name)

	serviceAccount, err = c.kubeClient.ServiceAccounts("namespace2").Get("default")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(serviceAccount.ImagePullSecrets))
	assert.Equal(t, *argDefaultSecretName, serviceAccount.ImagePullSecrets[0].Name)
}

func TestProcessWithExistingSecrets(t *testing.T) {
	kubeClient := newFakeKubeClient()
	ecrClient := newFakeEcrClient()

	c := &controller{kubeClient, ecrClient}

	secret := &api.Secret{
		ObjectMeta: api.ObjectMeta{
			Name: *argDefaultSecretName,
		},
		Data: map[string][]byte{
			".dockerconfigjson": []byte("some other config"),
		},
		Type: "some other type",
	}

	_, err := c.kubeClient.Secrets("namespace1").Create(secret)
	assert.Nil(t, err)
	_, err = c.kubeClient.Secrets("namespace2").Create(secret)
	assert.Nil(t, err)

	err = c.process()
	assert.Nil(t, err)

	secret, err = c.kubeClient.Secrets("namespace1").Get(*argDefaultSecretName)
	assert.Nil(t, err)
	assert.Equal(t, *argDefaultSecretName, secret.Name)
	assert.Equal(t, map[string][]byte{
		".dockerconfigjson": []byte(fmt.Sprintf(dockerJSONTemplate, "fakeEndpoint", "fakeToken")),
	}, secret.Data)
	assert.Equal(t, secret.Type, api.SecretType("kubernetes.io/dockerconfigjson"))

	secret, err = c.kubeClient.Secrets("namespace2").Get(*argDefaultSecretName)
	assert.Nil(t, err)
	assert.Equal(t, *argDefaultSecretName, secret.Name)
	assert.Equal(t, map[string][]byte{
		".dockerconfigjson": []byte(fmt.Sprintf(dockerJSONTemplate, "fakeEndpoint", "fakeToken")),
	}, secret.Data)
	assert.Equal(t, api.SecretType("kubernetes.io/dockerconfigjson"), secret.Type)
}

func TestProcessNoDefaultServiceAccount(t *testing.T) {
	kubeClient := newFakeKubeClient()
	ecrClient := newFakeEcrClient()

	c := &controller{kubeClient, ecrClient}

	err := c.kubeClient.ServiceAccounts("namespace1").Delete("default")
	assert.Nil(t, err)
	err = c.kubeClient.ServiceAccounts("namespace2").Delete("default")
	assert.Nil(t, err)

	err = c.process()
	assert.NotNil(t, err)
}

func TestProcessWithExistingImagePullSecrets(t *testing.T) {
	kubeClient := newFakeKubeClient()
	ecrClient := newFakeEcrClient()

	c := &controller{kubeClient, ecrClient}

	serviceAccount, err := c.kubeClient.ServiceAccounts("namespace1").Get("default")
	assert.Nil(t, err)
	serviceAccount.ImagePullSecrets = append(serviceAccount.ImagePullSecrets, api.LocalObjectReference{Name: "someOtherSecret"})
	_, err = c.kubeClient.ServiceAccounts("namespace1").Update(serviceAccount)

	serviceAccount, err = c.kubeClient.ServiceAccounts("namespace2").Get("default")
	assert.Nil(t, err)
	serviceAccount.ImagePullSecrets = append(serviceAccount.ImagePullSecrets, api.LocalObjectReference{Name: "someOtherSecret"})
	_, err = c.kubeClient.ServiceAccounts("namespace2").Update(serviceAccount)

	c.process()

	serviceAccount, err = c.kubeClient.ServiceAccounts("namespace1").Get("default")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(serviceAccount.ImagePullSecrets))
	assert.Equal(t, "someOtherSecret", serviceAccount.ImagePullSecrets[0].Name)
	assert.Equal(t, *argDefaultSecretName, serviceAccount.ImagePullSecrets[1].Name)

	serviceAccount, err = c.kubeClient.ServiceAccounts("namespace2").Get("default")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(serviceAccount.ImagePullSecrets))
	assert.Equal(t, "someOtherSecret", serviceAccount.ImagePullSecrets[0].Name)
	assert.Equal(t, *argDefaultSecretName, serviceAccount.ImagePullSecrets[1].Name)
}

func TestDefaultAwsRegionFromArgs(t *testing.T) {
	assert.Equal(t, "us-east-1", *argAWSRegion)
}

func TestAwsRegionFromEnv(t *testing.T) {
	expectedRegion := "us-steve-1"

	os.Setenv("awsaccount", "12345678")
	os.Setenv("awsregion", expectedRegion)
	validateParams()

	assert.Equal(t, expectedRegion, *argAWSRegion)
}
