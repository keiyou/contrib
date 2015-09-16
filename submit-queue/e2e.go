/*
Copyright 2015 The Kubernetes Authors All rights reserved.

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

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	github_util "k8s.io/contrib/github"
	"k8s.io/contrib/submit-queue/jenkins"

	"github.com/golang/glog"
	github_api "github.com/google/go-github/github"
)

type ExternalState struct {
	// exported so that the json marshaller will print them
	CurrentPR   *github_api.PullRequest
	Message     []string
	Err         error
	BuildStatus map[string]string
	Whitelist   []string
}

type e2eTester struct {
	sync.Mutex
	state       *ExternalState
	BuildStatus map[string]string
	Config      *SubmitQueueConfig
}

func (e *e2eTester) msg(msg string, args ...interface{}) {
	e.Lock()
	defer e.Unlock()
	if len(e.state.Message) > 50 {
		e.state.Message = e.state.Message[1:]
	}
	expanded := fmt.Sprintf(msg, args...)
	e.state.Message = append(e.state.Message, fmt.Sprintf("%v: %v", time.Now().UTC(), expanded))
	glog.V(2).Info(expanded)
}

func (e *e2eTester) error(err error) {
	e.Lock()
	defer e.Unlock()
	e.state.Err = err
}

func (e *e2eTester) locked(f func()) {
	e.Lock()
	defer e.Unlock()
	f()
}

func (e *e2eTester) setBuildStatus(build, status string) {
	e.locked(func() { e.BuildStatus[build] = status })
}

func (e *e2eTester) checkBuilds() (allStable bool) {
	// Test if the build is stable in Jenkins
	jenkinsClient := &jenkins.JenkinsClient{Host: e.Config.JenkinsHost}
	allStable = true
	for _, build := range e.Config.JenkinsJobs {
		e.msg("Checking build stability for %s", build)
		stable, err := jenkinsClient.IsBuildStable(build)
		if err != nil {
			e.msg("Error checking build %v: %v", build, err)
			e.setBuildStatus(build, "Error checking: "+err.Error())
			allStable = false
			continue
		}
		if stable {
			e.setBuildStatus(build, "Stable")
		} else {
			e.setBuildStatus(build, "Not Stable")
		}
	}
	return allStable
}

func (e *e2eTester) waitForStableBuilds() {
	for !e.checkBuilds() {
		e.msg("Not all builds stable. Checking again in 30s")
		time.Sleep(30 * time.Second)
	}
}

// This is called on a potentially mergeable PR
func (e *e2eTester) runE2ETests(pr *github_api.PullRequest, issue *github_api.Issue) error {
	e.locked(func() { e.state.CurrentPR = pr })
	defer e.locked(func() { e.state.CurrentPR = nil })
	e.msg("Considering PR %d", *pr.Number)

	e.waitForStableBuilds()

	// if there is a 'e2e-not-required' label, just merge it.
	if len(e.Config.DontRequireE2ELabel) > 0 && github_util.HasLabel(issue.Labels, e.Config.DontRequireE2ELabel) {
		e.msg("Merging %d since %s is set", *pr.Number, e.Config.DontRequireE2ELabel)
		return e.Config.MergePR(*pr.Number, "submit-queue")
	}

	body := "@k8s-bot test this [submit-queue is verifying that this PR is safe to merge]"
	if err := e.Config.WriteComment(*pr.Number, body); err != nil {
		e.error(err)
		return err
	}

	// Wait for the build to start
	err := e.Config.WaitForPending(*pr.Number)

	// Wait for the status to go back to 'success'
	ok, err := e.Config.ValidateStatus(*pr.Number, []string{}, true)
	if err != nil {
		e.error(err)
		return err
	}
	if !ok {
		e.msg("Status after build is not 'success', skipping PR %d", *pr.Number)
		return nil
	}
	return e.Config.MergePR(*pr.Number, "submit-queue")
}

func (e *e2eTester) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var (
		data []byte
		err  error
	)
	e.locked(func() {
		if e.state != nil {
			data, err = json.MarshalIndent(e.state, "", "\t")
		} else {
			data = []byte("{}")
		}
	})

	res.Header().Set("Content-type", "application/json")
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(err.Error()))
	} else {
		res.WriteHeader(http.StatusOK)
		res.Write(data)
	}
}
