{{ template "header" .}}
{{ template "campaignheader" .}}

<form class="form" method=post action=campaign>
<input type=hidden name="action" value="insert" />

<h3>修改活动推广计划</h3>
<div class="table-responsive">

<table class="table table-striped table-sm">

<tr><td>活动名称:</td><td><input type=text name=campaign_name size=40 /></td></tr>
<tr><td>活动 ID: </td><td><input type=text name=foreign_id size=10 /></td></tr>

<tr><td valign=top>频次控制:</td><td>
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
<tr><td>最大广告位个数: </td><td><input type=text name=page_cap size=1 value=2> (当前页面最多允许放置广告位个数)</td></tr>

<tr><td>活动质量:</td><td>{{range $key, $val := .Other.campaigns }}
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
