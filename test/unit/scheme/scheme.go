package scheme

import (
	rookapis "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	apiextenstionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"

	csiopapi "github.com/ceph/ceph-csi-operator/api/v1"

	"github.com/Mirantis/pelagia/pkg/apis"
	lcmv1alpha1 "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1"
)

var Scheme = runtime.NewScheme()
var Codecs = serializer.NewCodecFactory(Scheme)
var SchemeBuilder = runtime.SchemeBuilder{
	apis.AddToScheme,
	k8sscheme.AddToScheme,
	rookapis.AddToScheme,
	apiextenstionsv1.AddToScheme,
	lcmv1alpha1.AddToScheme,
	csiopapi.AddToScheme,
}
var Encoder = json.NewSerializerWithOptions(json.DefaultMetaFactory, Scheme, Scheme, json.SerializerOptions{Yaml: true})

func init() {
	err := SchemeBuilder.AddToScheme(Scheme)
	if err != nil {
		panic(err)
	}
}

func Decode(yaml []byte) (runtime.Object, error) {
	return runtime.Decode(Codecs.UniversalDeserializer(), yaml)
}

func MustDecode(yaml []byte) runtime.Object {
	obj, err := Decode(yaml)
	if err != nil {
		panic(err)
	}
	return obj
}

func Encode(obj runtime.Object) ([]byte, error) {
	return runtime.Encode(Encoder, obj)
}
