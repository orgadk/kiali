package virtual_services

import (
	"testing"

	"github.com/kiali/kiali/kubernetes"
	"github.com/kiali/kiali/services/models"
	"github.com/kiali/kiali/tests/data"
	"github.com/stretchr/testify/assert"
)

func TestOneVirtualServicePerHost(t *testing.T) {
	vss := []kubernetes.IstioObject{
		buildVirtualService("virtual-1", "reviews"),
		buildVirtualService("virtual-2", "ratings"),
	}
	validations := SingleHostChecker{
		Namespace:       "bookinfo",
		VirtualServices: vss,
	}.Check()

	noValidationResult(t, validations)
}

func TestOneVirtualServicePerFQDNHost(t *testing.T) {
	vss := []kubernetes.IstioObject{
		buildVirtualService("virtual-1", "reviews.bookinfo.svc.cluster.local"),
		buildVirtualService("virtual-2", "ratings.bookinfo.svc.cluster.local"),
	}
	validations := SingleHostChecker{
		Namespace:       "bookinfo",
		VirtualServices: vss,
	}.Check()

	noValidationResult(t, validations)
}

func TestOneVirtualServicePerFQDNWildcardHost(t *testing.T) {
	vss := []kubernetes.IstioObject{
		buildVirtualService("virtual-1", "*.bookinfo.svc.cluster.local"),
		buildVirtualService("virtual-2", "*.eshop.svc.cluster.local"),
	}
	validations := SingleHostChecker{
		Namespace:       "bookinfo",
		VirtualServices: vss,
	}.Check()

	noValidationResult(t, validations)
}

func TestRepeatingSimpleHost(t *testing.T) {
	vss := []kubernetes.IstioObject{
		buildVirtualService("virtual-1", "reviews"),
		buildVirtualService("virtual-2", "reviews"),
	}

	validations := SingleHostChecker{
		Namespace:       "bookinfo",
		VirtualServices: vss,
	}.Check()

	presentValidationTest(t, validations, "virtual-1")
	presentValidationTest(t, validations, "virtual-2")
}

func TestRepeatingFQDNHost(t *testing.T) {
	vss := []kubernetes.IstioObject{
		buildVirtualService("virtual-1", "reviews.bookinfo.svc.cluster.local"),
		buildVirtualService("virtual-2", "reviews.bookinfo.svc.cluster.local"),
	}
	validations := SingleHostChecker{
		Namespace:       "bookinfo",
		VirtualServices: vss,
	}.Check()

	presentValidationTest(t, validations, "virtual-2")
}

func TestRepeatingFQDNWildcardHost(t *testing.T) {
	vss := []kubernetes.IstioObject{
		buildVirtualService("virtual-1", "*.bookinfo.svc.cluster.local"),
		buildVirtualService("virtual-2", "*.bookinfo.svc.cluster.local"),
	}
	validations := SingleHostChecker{
		Namespace:       "bookinfo",
		VirtualServices: vss,
	}.Check()

	presentValidationTest(t, validations, "virtual-1")
	presentValidationTest(t, validations, "virtual-2")
}

func TestIncludedIntoWildCard(t *testing.T) {
	vss := []kubernetes.IstioObject{
		buildVirtualService("virtual-1", "*.bookinfo.svc.cluster.local"),
		buildVirtualService("virtual-2", "reviews.bookinfo.svc.cluster.local"),
	}
	validations := SingleHostChecker{
		Namespace:       "bookinfo",
		VirtualServices: vss,
	}.Check()

	presentValidationTest(t, validations, "virtual-1")
	presentValidationTest(t, validations, "virtual-2")

	// Same test, with different order of appearance
	vss = []kubernetes.IstioObject{
		buildVirtualService("virtual-1", "reviews.bookinfo.svc.cluster.local"),
		buildVirtualService("virtual-2", "*.bookinfo.svc.cluster.local"),
	}
	validations = SingleHostChecker{
		Namespace:       "bookinfo",
		VirtualServices: vss,
	}.Check()

	presentValidationTest(t, validations, "virtual-2")
}

func TestShortHostNameIncludedIntoWildCard(t *testing.T) {
	vss := []kubernetes.IstioObject{
		buildVirtualService("virtual-1", "*.bookinfo.svc.cluster.local"),
		buildVirtualService("virtual-2", "reviews"),
	}
	validations := SingleHostChecker{
		Namespace:       "bookinfo",
		VirtualServices: vss,
	}.Check()

	presentValidationTest(t, validations, "virtual-1")
	presentValidationTest(t, validations, "virtual-2")
}

func TestMultipleHostsFailing(t *testing.T) {
	vss := []kubernetes.IstioObject{
		buildVirtualService("virtual-1", "reviews"),
		buildVirtualServiceMultipleHosts("virtual-2", []string{"reviews",
			"mongo.backup.svc.cluster.local", "mongo.staging.svc.cluster.local"}),
	}
	validations := SingleHostChecker{
		Namespace:       "bookinfo",
		VirtualServices: vss,
	}.Check()

	presentValidationTest(t, validations, "virtual-1")
	presentValidationTest(t, validations, "virtual-2")
}

func TestMultipleHostsPassing(t *testing.T) {
	vss := []kubernetes.IstioObject{
		buildVirtualService("virtual-1", "reviews"),
		buildVirtualServiceMultipleHosts("virtual-2", []string{"ratings",
			"mongo.backup.svc.cluster.local", "mongo.staging.svc.cluster.local"}),
	}
	validations := SingleHostChecker{
		Namespace:       "bookinfo",
		VirtualServices: vss,
	}.Check()

	noValidationResult(t, validations)
}

func buildVirtualService(name, host string) kubernetes.IstioObject {
	return buildVirtualServiceMultipleHosts(name, []string{host})
}

func buildVirtualServiceMultipleHosts(name string, hosts []string) kubernetes.IstioObject {
	return data.CreateEmptyVirtualService(name, "bookinfo", hosts)
}

func noValidationResult(t *testing.T, validations models.IstioValidations) {
	assert := assert.New(t)
	assert.Empty(validations)

	validation, ok := validations[models.IstioValidationKey{"virtualservice", "reviews"}]
	assert.False(ok)
	assert.Nil(validation)
}

func presentValidationTest(t *testing.T, validations models.IstioValidations, serviceName string) {
	assert := assert.New(t)
	assert.NotEmpty(validations)

	validation, ok := validations[models.IstioValidationKey{"virtualservice", serviceName}]
	assert.True(ok)

	assert.True(validation.Valid)
	assert.NotEmpty(validation.Checks)
	assert.Equal("warning", validation.Checks[0].Severity)
	assert.Equal("More than one Virtual Service for same host", validation.Checks[0].Message)
	assert.Equal("spec/hosts", validation.Checks[0].Path)
}
