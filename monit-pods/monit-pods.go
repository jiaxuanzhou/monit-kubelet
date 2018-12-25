package monit_pods

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"k8s.io/api/core/v1"
)

const (
	KubeletPodsUri = "http://127.0.0.1:10255/pods"
)

func GetPodsFromKubelet(hc *http.Client) (*v1.PodList, error) {
	res, err := hc.Get(KubeletPodsUri)
	if err != nil {
		return nil, err
	}
	var pl v1.PodList
	if res.Body != nil {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(body, &pl)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal podlist, %v", err)
		}
	}
	return &pl, nil
}

func CheckPodsStatus(pl *v1.PodList) ([]string, string) {
	unhealthyPods := []string{}
	for _, pod := range pl.Items {
		for _, containerState := range pod.Status.ContainerStatuses {
			if !containerState.Ready && containerState.State.Terminated != nil {
				unhealthyPods = append(unhealthyPods, pod.Name)
			}
		}
	}

	hostName, _ := os.Hostname()
	return unhealthyPods, hostName
}

func LogUnhealthyPods(hc *http.Client) {
	pl, err := GetPodsFromKubelet(hc)
	if err != nil {
		log.Printf("[ERROR] failed to list pods, err %v", err)
	}

	uPods, host := CheckPodsStatus(pl)
	if len(uPods) == 0 {
		log.Printf("[NORMAL] pods on node %s are all healthy.", host)
		return
	}
	log.Printf("pods %v on host %s are unhealthy!", uPods, host)
}
