/*
Copyright 2018 Pressinfra SRL.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package wordpress

import (
	"fmt"
	"path"

	"github.com/cooleo/slugify"
	"k8s.io/apimachinery/pkg/labels"

	wordpressv1alpha1 "github.com/presslabs/wordpress-operator/pkg/apis/wordpress/v1alpha1"
)

// Wordpress embeds wordpressv1alpha1.Wordpress and adds utility functions
type Wordpress struct {
	*wordpressv1alpha1.Wordpress
}

type component struct {
	name       string // eg. web, database, cache
	objNameFmt string
	objName    string
}

var (
	// WordpressSecret component
	WordpressSecret = component{name: "web", objNameFmt: "%s-wp"}
	// WordpressDeployment component
	WordpressDeployment = component{name: "web", objNameFmt: "%s"}
	// WordpressCron component
	WordpressCron = component{name: "cron", objNameFmt: "%s-wp-cron"}
	// WordpressDBUpgrade component
	WordpressDBUpgrade = component{name: "upgrade", objNameFmt: "%s-upgrade"}
	// WordpressService component
	WordpressService = component{name: "web", objNameFmt: "%s"}
	// WordpressIngress component
	WordpressIngress = component{name: "web", objNameFmt: "%s"}
	// WordpressCodePVC component
	WordpressCodePVC = component{name: "code", objNameFmt: "%s-code"}
	// WordpressMediaPVC component
	WordpressMediaPVC = component{name: "media", objNameFmt: "%s-media"}
)

// New wraps a wordpressv1alpha1.Wordpress into a Wordpress object
func New(obj *wordpressv1alpha1.Wordpress) *Wordpress {
	return &Wordpress{obj}
}

// Unwrap returns the wrapped wordpressv1alpha1.Wordpress object
func (o *Wordpress) Unwrap() *wordpressv1alpha1.Wordpress {
	return o.Wordpress
}

// Labels returns default label set for wordpressv1alpha1.Wordpress
func (o *Wordpress) Labels() labels.Set {
	partOf := "wordpress"
	if o.ObjectMeta.Labels != nil && len(o.ObjectMeta.Labels["app.kubernetes.io/part-of"]) > 0 {
		partOf = o.ObjectMeta.Labels["app.kubernetes.io/part-of"]
	}

	labels := labels.Set{
		"app.kubernetes.io/name":     "wordpress",
		"app.kubernetes.io/part-of":  partOf,
		"app.kubernetes.io/instance": o.ObjectMeta.Name,
	}

	return labels
}

// ComponentLabels returns labels for a label set for a wordpressv1alpha1.Wordpress component
func (o *Wordpress) ComponentLabels(component component) labels.Set {
	l := o.Labels()
	l["app.kubernetes.io/component"] = component.name

	if component == WordpressDBUpgrade {
		l["wordpress.presslabs.org/upgrade-for"] = o.ImageVersion()
	}

	return l
}

// ComponentName returns the object name for a component
func (o *Wordpress) ComponentName(component component) string {
	name := component.objName
	if len(component.objNameFmt) > 0 {
		name = fmt.Sprintf(component.objNameFmt, o.ObjectMeta.Name)
	}

	if component == WordpressDBUpgrade {
		name = fmt.Sprintf("%s-for-%s", name, o.ImageVersion())
	}

	return name
}

// ImageVersion returns the version from the image in a format suitable
// for kubernetes object names and labels
func (o *Wordpress) ImageVersion() string {
	return slugify.Slugify(o.Spec.Image)
}

// WebPodLabels return labels to apply to web pods
func (o *Wordpress) WebPodLabels() labels.Set {
	l := o.Labels()
	l["app.kubernetes.io/component"] = "web"
	return l
}

// JobPodLabels return labels to apply to cli job pods
func (o *Wordpress) JobPodLabels() labels.Set {
	l := o.Labels()
	l["app.kubernetes.io/component"] = "wp-cli"
	return l
}

// MainDomain returns the site main domain or a local domain <cluster-name>.<namespace>.svc.cluster.local
func (o *Wordpress) MainDomain() string {
	if len(o.Spec.Routes) > 0 {
		return o.Spec.Routes[0].Domain
	}

	// return the local cluster name that points to wordpress service
	return fmt.Sprintf("%s.%s.svc", o.ComponentName(WordpressService), o.Namespace)
}

// HomeURL returns the WP_HOMEURL (e.g. http://example.com/)
func (o *Wordpress) HomeURL(subPaths ...string) string {
	scheme := "http"
	if len(o.Spec.TLSSecretRef) > 0 {
		scheme = "https"
	}

	paths := []string{"/"}
	if len(o.Spec.Routes) > 0 {
		paths = append(paths, o.Spec.Routes[0].Path)
	}
	paths = append(paths, subPaths...)
	p := path.Join(paths...)
	if p == "/" {
		p = ""
	}

	return fmt.Sprintf("%s://%s%s", scheme, o.MainDomain(), p)
}
