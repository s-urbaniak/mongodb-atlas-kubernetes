package observability

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoretry "k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	kubecli "github.com/mongodb/mongodb-atlas-kubernetes/v2/test/helper/e2e/k8s"
	akoretry "github.com/mongodb/mongodb-atlas-kubernetes/v2/test/helper/retry"
)

var defaultBackOff = wait.Backoff{
	Duration: 2 * time.Second,
	Factor:   1.0,
	Steps:    30 * 5,
}

func InstallLocalDevelopment(logger io.Writer, snapshotURL string) error {
	if err := os.Chdir("test/helper/observability"); err != nil {
		return fmt.Errorf("error changing directory: %w", err)
	}

	for _, cmdArgs := range [][]string{
		{"helm", "repo", "add", "prometheus-community", "https://prometheus-community.github.io/helm-charts"},
		{"helm", "repo", "add", "grafana", "https://grafana.github.io/helm-charts"},
		{"helm", "repo", "update"},
		{"helm", "upgrade", "--values", "kube-prometheus-helm.yaml", "--install", "kube-prometheus", "prometheus-community/kube-prometheus-stack", "-n", "monitoring", "--create-namespace"},
		{"helm", "upgrade", "--values", "loki-helm.yaml", "--install", "loki", "grafana/loki", "-n", "loki", "--create-namespace"},
		{"kubectl", "apply", "-f", "nodeports.yaml"},
		{"kubectl", "apply", "-f", "grafana-config.yaml"},
		{"kubectl", "-n", "monitoring", "delete", "--ignore-not-found", "configmap", "artifact-urls"},
		{"kubectl", "-n", "monitoring", "create", "configmap", "artifact-urls",
			fmt.Sprintf(`--from-literal=PROMETHEUS_SNAPSHOT_URL='%v/prometheus.tar.gz'`, snapshotURL),
		},
		{"kubectl", "apply", "--server-side", "--force-conflicts", "-f", "prometheus-local.yaml"},
		{"kubectl", "-n", "loki", "delete", "--ignore-not-found", "configmap", "artifact-urls"},
		{"kubectl", "-n", "loki", "create", "configmap", "artifact-urls",
			fmt.Sprintf(`--from-literal=LOKI_SNAPSHOT_URL='%v/loki.tar.gz'`, snapshotURL),
		},
		{"kubectl", "-n", "loki", "scale", "--replicas=0", "sts/loki"},
		{"kubectl", "apply", "--server-side", "--force-conflicts", "-f", "loki-sts-local.yaml"},
		{"kubectl", "-n", "loki", "scale", "--replicas=1", "sts/loki"},
		{"kubectl", "-n", "loki", "wait", "pods", "-l", `app.kubernetes.io/name=loki`, "--for", "condition=Ready", "--timeout=600s"},
		{"kubectl", "-n", "monitoring", "wait", "pods", "-l", `app.kubernetes.io/instance=kube-prometheus-kube-prome-prometheus`, "--for", "condition=Ready", "--timeout=600s"},
		// flush loki, as it was disconnected.
		{"curl", "-XPOST", "-v", `http://localhost:30002/flush`},
	} {
		if err := execCommand(logger, cmdArgs...); err != nil {
			return err
		}
	}

	return nil
}

func Install(logger io.Writer) error {
	if err := os.Chdir("test/helper/observability"); err != nil {
		return fmt.Errorf("error changing directory: %w", err)
	}

	ctx := context.Background()
	k8sClient, err := kubecli.CreateNewClient()
	if err != nil {
		return fmt.Errorf("error creating k8s client: %w", err)
	}

	for _, cmdArgs := range [][]string{
		{"helm", "repo", "add", "prometheus-community", "https://prometheus-community.github.io/helm-charts"},
		{"helm", "repo", "add", "grafana", "https://grafana.github.io/helm-charts"},
		{"helm", "repo", "update"},
		{"helm", "upgrade", "--values", "kube-prometheus-helm.yaml", "--install", "kube-prometheus", "prometheus-community/kube-prometheus-stack", "-n", "monitoring", "--create-namespace"},
		{"helm", "upgrade", "--values", "loki-helm.yaml", "--install", "loki", "grafana/loki", "-n", "loki", "--create-namespace"},
		{"helm", "upgrade", "--values", "promtail-helm.yaml", "--install", "promtail", "grafana/promtail", "-n", "promtail", "--create-namespace"},
		{"kubectl", "apply", "-f", "nodeports.yaml"},
		{"kubectl", "apply", "-f", "grafana-config.yaml"},
		{"kubectl", "apply", "--server-side", "--force-conflicts", "-f", "prometheus.yaml"},
	} {
		if err := execCommand(logger, cmdArgs...); err != nil {
			return err
		}
	}

	err = retry(ctx, func() error {
		return execCommand(logger, "kubectl", "-n", "monitoring", "scale", "--replicas=0", "deployment/kube-prometheus-kube-state-metrics")
	})
	if err != nil {
		return fmt.Errorf("error executing command: %w", err)
	}

	err = PatchKubeStateMetricsDeployment(ctx, k8sClient)
	if err != nil {
		return fmt.Errorf("error patching kube state metrics: %w", err)
	}

	for _, cmdArgs := range [][]string{
		{"kubectl", "-n", "monitoring", "scale", "--replicas=1", "deployment/kube-prometheus-kube-state-metrics"},
		{"kubectl", "apply", "--server-side", "-f", "ksm-crb.yaml"},
		{"kubectl", "-n", "loki", "rollout", "status", "--watch", "statefulset/loki"},
		{"kubectl", "-n", "promtail", "rollout", "status", "--watch", "deployment/promtail"},
		{"kubectl", "-n", "monitoring", "rollout", "status", "--watch", "deployment/kube-prometheus-kube-state-metrics"},
		{"kubectl", "-n", "monitoring", "rollout", "status", "--watch", "statefulset/prometheus-kube-prometheus-kube-prome-prometheus"},
	} {
		if err := execCommand(logger, cmdArgs...); err != nil {
			return err
		}
	}
	return nil
}

func execCommand(logger io.Writer, cmdArgs ...string) error {
	fmt.Fprintln(logger, cmdArgs)
	out, err := exec.Command(cmdArgs[0], cmdArgs[1:]...).Output()
	fmt.Fprintln(logger, string(out))

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		logger.Write(exitErr.Stderr)
	}

	if err != nil {
		return fmt.Errorf("error executing command: %w", err)
	}
	return nil
}

func readUnstructeredObject(filename string) (*unstructured.Unstructured, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading %q: %w", filename, err)
	}

	var obj map[string]interface{}
	if err := yaml.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("error unmarshalling %q: %w", filename, err)
	}

	return &unstructured.Unstructured{Object: obj}, nil
}

func PatchKubeStateMetricsDeployment(ctx context.Context, k8s client.Client) error {
	ksmConfig, err := readUnstructeredObject("ksm-config.yaml")
	if err != nil {
		return err
	}
	err = k8s.Patch(ctx, ksmConfig, client.Apply, client.ForceOwnership, client.FieldOwner("ako-test"))
	if err != nil {
		return fmt.Errorf("error patching ksm config: %w", err)
	}

	_, err = akoretry.RetryUpdateOnConflict(ctx, k8s, k8stypes.NamespacedName{Namespace: "monitoring", Name: "kube-prometheus-kube-state-metrics"}, func(ksm *v1.Deployment) {
		// TODO(sur): submit kube-prometheus upstream PR to bump ksm
		ksm.Spec.Template.Spec.Containers[0].Image = "registry.k8s.io/kube-state-metrics/kube-state-metrics:v2.12.0"
		if len(ksm.Spec.Template.Spec.Containers[0].VolumeMounts) == 0 {
			ksm.Spec.Template.Spec.Containers[0].Args = append(ksm.Spec.Template.Spec.Containers[0].Args, "--custom-resource-state-config-file=/etc/kube-state-metrics/ako.yaml")
			ksm.Spec.Template.Spec.Containers[0].VolumeMounts = append(ksm.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
				Name:      "ako",
				MountPath: "/etc/kube-state-metrics",
			})
			ksm.Spec.Template.Spec.Volumes = append(ksm.Spec.Template.Spec.Volumes, corev1.Volume{
				Name: "ako",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: "kube-state-metrics-config"},
						Items:                []corev1.KeyToPath{{Key: "ako", Path: "ako.yaml"}},
					},
				},
			})
		}
	})
	if err != nil {
		return fmt.Errorf("error updating ksm deployment: %w", err)
	}

	return nil
}

func retry(ctx context.Context, f func() error) error {
	return clientgoretry.OnError(
		defaultBackOff, func(err error) bool { return true },
		func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			err := f()
			if err != nil {
				fmt.Println(err)
			}
			return err
		},
	)
}
