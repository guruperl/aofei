{{ template "header" .}}
{{ template "balanceheader" .}}

{{if .ARGS.total_balance_id}}
<form name=total class="form" method=post action="balance">
<input type=hidden name=action value="update" />
<input type=hidden name=entitytype_id value="{{ index .ARGS.entitytype_id 0}}" />
<input type=hidden name=campaign_id value="{{index .ARGS.campaign_id 0}}" />
<input type=hidden name=campaign_md5 value="{{index .ARGS.campaign_md5 0}}" />
<input type=hidden name=campaign_name value="{{index .ARGS.campaign_name 0}}" />{{ if eq "42" (index .ARGS.entitytype_id 0)}}
<input type=hidden name=item_id value="{{index .ARGS.item_id 0}}" />
<input type=hidden name=item_md5 value="{{index .ARGS.item_md5 0}}" />
<input type=hidden name=item_name value="{{index .ARGS.item_name 0}}" />{{end}}

<h3>总花费</h3>
<div class="table-responsive">{{ with .Lists }}{{ range . }}{{if eq "total_balance_id" .which}}
<table class="table table-striped table-sm">
<thead> <tr> <th> </th> <th>预算</th> <th>曝光量</th> <th>点击量</th> <th> </th> </tr> </thead>
<tbody> <tr>
<td>预计值: </td>
<td><input type=text name=limit_spend value="{{ .limit_spend}}" /></td>
<td><input type=text name=limit_imp value="{{ .limit_imp}}" /></td>
<td><input type=text name=limit_cli value="{{ .limit_cli}}" /></td>
<td><input type=hidden name=balance_id value={{.balance_id}} /><input type=submit value="保存" /></td>
</tr> <tr>
<td>实际值: </td>
<td>{{ .current_spend}}</td>
<td>{{ .current_imp}}</td>
<td>{{ .current_cli}}</td>
<td></td>
</tr> </tobdy>
</table>{{end}}{{end}}{{end}}
</div>
</form>
{{else}}
<h4><a href="balance?action=startnew&entitytype_id={{index .ARGS.entitytype_id 0}}&which=total_balance_id&campaign_id={{index .ARGS.campaign_id 0}}&campaign_md5={{index .ARGS.campaign_md5 0}}&campaign_name={{index .ARGS.campaign_name 0 | urlquery}}{{if eq "42" (index .ARGS.entitytype_id 0)}}&item_id={{index .ARGS.item_id 0}}&item_md5={{index .ARGS.item_md5 0}}&item_name={{index .ARGS.item_name 0 | urlquery}}{{end}}">增加总预算</a></h4>
{{end}}

{{if .ARGS.daily_balance_id}}
<form name=daily class="form" method=post action="balance">
<input type=hidden name=action value="update" />
<input type=hidden name=entitytype_id value="{{ index .ARGS.entitytype_id 0}}" />
<input type=hidden name=campaign_id value="{{index .ARGS.campaign_id 0}}" />
<input type=hidden name=campaign_md5 value="{{index .ARGS.campaign_md5 0}}" />
<input type=hidden name=campaign_name value="{{index .ARGS.campaign_name 0 | urlquery}}" />{{if eq "42" (index .ARGS.entitytype_id 0) }}
<input type=hidden name=item_id value="{{index .ARGS.item_id 0}}" />
<input type=hidden name=item_md5 value="{{index .ARGS.item_md5 0}}" />
<input type=hidden name=item_name value="{{index .ARGS.item_name 0}}" />{{end}}

<h3>日预算</h3>
<div class="table-responsive">{{ with .Lists }}{{ range . }}{{if eq "daily_balance_id" .which}}
<table class="table table-striped table-sm">
<thead> <tr><th> </th> <th>预算</th> <th>曝光量</th> <th>点击量</th><th> </th> </tr> </thead>
<tbody> <tr>
<td>预计值</td>
<td><input type=text name=limit_spend value="{{ .limit_spend}}" /></td>
<td><input type=text name=limit_imp value="{{ .limit_imp}}" /></td>
<td><input type=text name=limit_cli value="{{ .limit_cli}}" /></td>
<td><input type=hidden name=balance_id value={{.balance_id}} /><input type=submit value="保存" /></td>
</tr> <tr>
<td>实际值:</td>
<td>{{ .current_spend}}</td>
<td>{{ .current_imp}}</td>
<td>{{ .current_cli}}</td>
<td> </td>
</tr> </tobdy>
</table>{{end}}{{end}}{{end}}
</div>
</form>
{{else}}
<h4><a href="balance?action=startnew&entitytype_id={{index .ARGS.entitytype_id 0}}&which=daily_balance_id&campaign_id={{index .ARGS.campaign_id 0}}&campaign_md5={{index .ARGS.campaign_md5 0}}&campaign_name={{index .ARGS.campaign_name 0 | urlquery}}{{if eq "42" (index .ARGS.entitytype_id 0)}}&item_id={{index .ARGS.item_id 0}}&item_md5={{index .ARGS.item_md5 0}}&item_name={{index .ARGS.item_name 0 | urlquery}}{{end}}">增加日预算</a></h4>
{{end}}

{{ template "footer" }}
