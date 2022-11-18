package main

var emailReportTemp string = `A user has reported the following post
ID: {{ .ID }}
Reporting User: {{ .ReportUser }}
Author: {{ .Post.Author }}
Post Date: {{ .Post.Time}}
Reason: {{ .Reason }}
From: {{ .IP }} [{{ .Hostname }}]
-------------------------
{{ .Text }}
-------------------------
`
