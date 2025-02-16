{{ template "header" .}}
{{ template "siteheader" .}}

<form class="form" action="site" method=post>
<input type=hidden name="action" value="insert">

<h2>Create New Site or App</h2>
<div class="table-responsive">
<table class="table table-striped table-sm">

<tr><td>Site Name:</td><td><input type=text name=site_name size=40></td></tr>
<tr><td>URL:</td><td><input type=text name=site_url size=40 value="http://"></td></tr>

<tr><td>Quality:</td><td>{{range $key, $val := .Other.sites }}
<tr><td>{{$key}}:</td><td><select size=1 name={{$key}}>{{range $k, $v := $val}}
<option value="{{$k}}">{{$v}}</option>{{end}}</td></tr>{{end}}

<tr><td>Accept Campaigns:</td><td>{{range $key, $val := .Other.campaigns }}
<tr><td>{{$key}}:</td><td><select size=1 name={{$key}}>{{range $k, $v := $val}}
<option value="{{$k}}">{{$v}}</option>{{end}}</td></tr>{{end}}

<tr><td colspan=2> &nbsp; </td><td>
</table>
<input type=submit value=" Add New Site " />
</form>

</div>

{{ template "footer" .}}
