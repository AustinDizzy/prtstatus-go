PRT Status server (with Go)
[![Build Status](https://travis-ci.org/AustinDizzy/prtstatus-go.svg)](https://travis-ci.org/AustinDizzy/prtstatus-go) [![GoDoc](https://godoc.org/github.com/AustinDizzy/prtstatus-go?status.svg)](https://godoc.org/github.com/AustinDizzy/prtstatus-go/prt)
=====

This repository hosts the server code for the WVU PRT Status (https://prtstat.us) project. The server is currently hosted on Google App Engine (https://wvuprtstatus-ec646.appspot.com/) and works well within Google's generous free tier.

The repository also has [a Go library](/prt/) for interacting with WVU's PRT Status API (which is just a simple net/http wrapper with custom types and other utility functions). See [GoDoc](https://godoc.org/github.com/AustinDizzy/prtstatus-go/prt) for more info.

#### Built With
This project is built with many different technologies, including (but not limited to) the following:
 - [PRT Status API](https://prtstatus.wvu.edu) - WVU's UR-Web offers an API which they use to pull PRT status on Portal and other WVU sites
 - [Go](http://golang.org) - a highly efficient and scalable language
 - [Google App Engine](https://cloud.google.com/appengine) - a perfect match for Go; a highly scalable and available PaaS
 - [Google Cloud Datastore](https://cloud.google.com/datastore/) - a fast and highly scalable database solution by Google
 - [Pure.css](https://purecss.io) - a minimal response CSS web framework by Yahoo
 - [Firebase Cloud Messaging](https://firebase.google.com/products/cloud-messaging/) - a realtime client messaging product by Google
 - [Pushbullet](https://pushbullet.com) - a notification sync service, being used to provide a [public notification channel](https://www.pushbullet.com/channel-popup?tag=wvuprtstatus) so iOS users can receive notifications
 - [Weather Underground](https://www.wunderground.com/) - Weather Underground's free API is used to collect weather information from the [weather station installed on the Brooks Hall green roof](https://www.wunderground.com/personal-weather-station/dashboard?ID=KWVMORGA25).


 #### Contributing
Want to contribute to the project? Go right ahead! Fork this repo and browse the issues to find something that needs a fixin', or build and implement a new feature. If your code is sound and your pull request is well written, I'll accept the PR and the CI service will automatically push your code into production.