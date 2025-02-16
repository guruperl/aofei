{{ template "header" .}}
{{ template "balanceheader" .}}

<!--<h3>Add {{if eq "total_balance_id" (index .ARGS.which 0)}}Total{{else}}Daily{{end}} Spending</h3>-->
<h3>新增{{if eq "total_balance_id" (index .ARGS.which 0)}}总{{else}}日{{end}}预算</h3>

<form name=total class="form" method=post action="balance">
<input type=hidden name=action value="insert" />
<input type=hidden name=which value="{{index .ARGS.which 0}}" />
<input type=hidden name=entitytype_id value="{{ index .ARGS.entitytype_id 0}}" />
<input type=hidden name=campaign_id value="{{index .ARGS.campaign_id 0}}" />
<input type=hidden name=campaign_md5 value="{{index .ARGS.campaign_md5 0}}" />
<input type=hidden name=campaign_name value="{{index .ARGS.campaign_name 0}}" />{{if eq "42" (index .ARGS.entitytype_id 0)}}
<input type=hidden name=item_id value="{{index .ARGS.item_id 0}}" />
<input type=hidden name=item_md5 value="{{index .ARGS.item_md5 0}}" />
<input type=hidden name=item_name value="{{index .ARGS.item_name 0}}" />{{end}}

<div class="table-responsive">
<table class="table table-striped table-sm">
<thead> <tr> <th>预算</th> <th>曝光次数</th> <th>点击次数</th> </tr> </thead>
<tbody> <tr>
<td><input type=text name=limit_spend size=8 /></td>
<td><input type=text name=limit_imp size=8 /></td>
<td><input type=text name=limit_cli size=8 /></td>
</tr> </tobdy>
</table>
</div>

<input type=submit value=" 保存 " />

</form>

{{ template "footer" }}
