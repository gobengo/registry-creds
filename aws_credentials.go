/*
Copyright (c) 2016, UPMC Enterprises
All rights reserved.
Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:
    * Redistributions of source code must retain the above copyright
      notice, this list of conditions and the following disclaimer.
    * Redistributions in binary form must reproduce the above copyright
      notice, this list of conditions and the following disclaimer in the
      documentation and/or other materials provided with the distribution.
    * Neither the name UPMC Enterprises nor the
      names of its contributors may be used to endorse or promote products
      derived from this software without specific prior written permission.
THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL UPMC ENTERPRISES BE LIABLE FOR ANY
DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
*/

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	flag "github.com/spf13/pflag"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned"
	kubectl_util "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

const (
	dockerJSONTemplate = `{"auths":{"%s":{"auth":"%s","email":"none"}}}`
)

var (
	flags                = flag.NewFlagSet("", flag.ContinueOnError)
	cluster              = flags.Bool("use-kubernetes-cluster-service", true, `If true, use the built in kubernetes cluster for creating the client`)
	argKubecfgFile       = flags.String("kubecfg-file", "", `Location of kubecfg file for access to kubernetes master service; --kube_master_url overrides the URL part of this; if neither this nor --kube_master_url are provided, defaults to service account tokens`)
	argKubeMasterURL     = flags.String("kube-master-url", "", `URL to reach kubernetes master. Env variables in this flag will be expanded.`)
	argDefaultSecretName = flags.String("default-secret-name", "awsecr-creds", `Default secret name`)
	argDefaultNamespace  = flags.String("default-namespace", "default", `Default namespace`)
	argAWSRegion         = flags.String("aws-region", "us-east-1", `Default AWS region`)
	argRefreshMinutes    = flags.Int("refresh-mins", 715, `Default time to wait before refreshing (11 hours 55 mins)`)
)

var (
	awsAccountID string
)

type controller struct {
	kubeClient kubeInterface
	ecrClient  ecrInterface
}

type kubeInterface interface {
	Secrets(namespace string) unversioned.SecretsInterface
	Namespaces() unversioned.NamespaceInterface
	ServiceAccounts(namespace string) unversioned.ServiceAccountsInterface
}

type ecrInterface interface {
	GetAuthorizationToken(input *ecr.GetAuthorizationTokenInput) (*ecr.GetAuthorizationTokenOutput, error)
}

func newEcrClient() ecrInterface {
	return ecr.New(session.New(), aws.NewConfig().WithRegion(*argAWSRegion))
}

func newKubeClient() kubeInterface {
	var kubeClient *unversioned.Client
	var config *restclient.Config
	var err error

	clientConfig := kubectl_util.DefaultClientConfig(flags)

	if *cluster {
		if kubeClient, err = unversioned.NewInCluster(); err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}
	} else {
		config, err = clientConfig.ClientConfig()
		if err != nil {
			log.Fatalf("error connecting to the client: %v", err)
		}
		kubeClient, err = unversioned.New(config)

		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}
	}

	return kubeClient
}

func (c *controller) getECRAuthorizationKey() (token *ecr.AuthorizationData, err error) {
	params := &ecr.GetAuthorizationTokenInput{
		RegistryIds: []*string{
			aws.String(awsAccountID),
		},
	}

	resp, err := c.ecrClient.GetAuthorizationToken(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	token = resp.AuthorizationData[0]

	return
}

func generateSecretObj(token *ecr.AuthorizationData) *api.Secret {
	dockerConfigJSONContent := fmt.Sprintf(dockerJSONTemplate, *token.ProxyEndpoint, *token.AuthorizationToken)

	secret := &api.Secret{
		ObjectMeta: api.ObjectMeta{
			Name: *argDefaultSecretName,
		},
		Data: map[string][]byte{
			".dockerconfigjson": []byte(dockerConfigJSONContent),
		},
		Type: "kubernetes.io/dockerconfigjson",
	}

	return secret
}

func (c *controller) process() error {
	newToken, _ := c.getECRAuthorizationKey()
	newSecret := generateSecretObj(newToken)

	// Get all namespaces
	namespaces, err := c.kubeClient.Namespaces().List(api.ListOptions{})
	if err != nil {
		return err
	}

	for _, namespace := range namespaces.Items {

		if namespace.GetName() == "kube-system" {
			continue
		}

		// Check if the secret exists for the namespace
		_, err := c.kubeClient.Secrets(namespace.GetName()).Get(*argDefaultSecretName)

		if err != nil {
			// Secret not found, create
			_, err := c.kubeClient.Secrets(namespace.GetName()).Create(newSecret)
			if err != nil {
				return err
			}
		} else {
			// Existing secret needs updated
			_, err := c.kubeClient.Secrets(namespace.GetName()).Update(newSecret)
			if err != nil {
				return err
			}
		}

		// Check if ServiceAccount exists
		serviceAccount, err := c.kubeClient.ServiceAccounts(namespace.GetName()).Get("default")

		if err != nil {
			return err
		}

		// Update existing one if image pull secrets already exists for aws ecr token
		imagePullSecretFound := false
		for i, imagePullSecret := range serviceAccount.ImagePullSecrets {
			if imagePullSecret.Name == *argDefaultSecretName {
				serviceAccount.ImagePullSecrets[i] = api.LocalObjectReference{Name: *argDefaultSecretName}
				imagePullSecretFound = true
				break
			}
		}

		// Append to list of existing service accounts if there isn't one already
		if !imagePullSecretFound {
			serviceAccount.ImagePullSecrets = append(serviceAccount.ImagePullSecrets, api.LocalObjectReference{Name: *argDefaultSecretName})
		}

		_, err = c.kubeClient.ServiceAccounts(namespace.GetName()).Update(serviceAccount)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateParams() {
	awsAccountID = os.Getenv("awsaccount")
	if len(awsAccountID) == 0 {
		log.Fatal("Missing awsAccountId!")
	}

	awsRegionEnv := os.Getenv("awsregion")

	if len(awsRegionEnv) > 0 {
		argAWSRegion = &awsRegionEnv
	}
}

func main() {
	log.Print("Starting up...")
	flags.Parse(os.Args)

	validateParams()

	log.Print("Using AWS Account: ", awsAccountID)
	log.Printf("Using AWS Region: %s", *argAWSRegion)
	log.Print("Refresh Interval (minutes): ", *argRefreshMinutes)

	ecrClient := newEcrClient()
	kubeClient := newKubeClient()

	c := &controller{kubeClient, ecrClient}

	tick := time.Tick(time.Duration(*argRefreshMinutes) * time.Minute)

	// Process once now, then wait for tick
	c.process()

	for {
		select {
		case <-tick:
			log.Print("Refreshing credentials...")
			if err := c.process(); err != nil {
				log.Fatalf("Failed to load ecr credentials: %v", err)
			}
		}
	}

}
