{{ template "header" .}}
{{ template "campaignheader" .}}

<form class="form" method=post action=campaign>
<input type=hidden name="action" value="insert" />

<h3>Create New Campaign</h3>
<div class="table-responsive">

<table class="table table-striped table-sm">

<tr><td>Campaign Name:</td><td><input type=text name=campaign_name size=40 /></td></tr>
<tr><td>Foreign ID: </td><td><input type=text name=foreign_id size=10 /></td></tr>

<tr><td valign=top>Frequency Caps:</td><td>
<table>
<tr><th>Type</th><th>Number</th><th>Period</th><th>Throttle</th></tr>
<tr><td>Impression</td>
<td><input type=text name=cpm_fc size=3></td>
<td><input type=text name=cpm_length size=6>min</td>
<td><input type=text name=cpm_throttle size=6>min</td></tr>
<tr><td>Clicks</td>
<td><input type=text name=cpc_fc size=3></td>
<td><input type=text name=cpc_length size=6>min</td>
<td></td></tr>
</table>
</td></tr>
<tr><td>Page Cap: </td><td><input type=text name=page_cap size=1 value=2> (allowed campaigns on one page)</td></tr>

<tr><td>Quality:</td><td>{{range $key, $val := .Other.campaigns }}
<tr><td>{{$key}}:</td><td><select size=1 name={{$key}}>{{range $k, $v := $val}}
<option value="{{$k}}">{{$v}}</option>{{end}}</td></tr>{{end}}

<tr><td>Accept Site:</td><td>{{range $key, $val := .Other.sites }}
<tr><td>{{$key}}:</td><td><select size=1 name={{$key}}>{{range $k, $v := $val}}
<option value="{{$k}}">{{$v}}</option>{{end}}</td></tr>{{end}}

<tr><td colspan=2> &nbsp; </td><td>
</table>
<input type=submit value="Add New Campaign" />
</form>

</div>


{{template "footer"}}
