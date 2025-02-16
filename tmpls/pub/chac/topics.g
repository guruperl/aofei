{{ template "header" .}}
{{ template "chacheader" .}}

<form class="form" method=post action="chac">
<input type=hidden name=action value="update" />
{{if eq "31" (index .ARGS.entitytype_id 0)}}
<input type=hidden name=site_id value="{{index .ARGS.site_id 0}}" />
<input type=hidden name=site_md5 value="{{index .ARGS.site_md5 0}}" />
<input type=hidden name=site_name value="{{index .ARGS.site_name 0}}" />
<input type=hidden name=entitytype_id value="31" />
<h3>{{index .ARGS.site_name 0}}</h3>{{else}}
<input type=hidden name=slot_id value="{{index .ARGS.slot_id 0}}" />
<input type=hidden name=slot_md5 value="{{index .ARGS.slot_md5 0}}" />
<input type=hidden name=slot_name value="{{index .ARGS.slot_name 0}}" />
<input type=hidden name=entitytype_id value="32" />
<h3>{{index .ARGS.slot_name 0}}</h3>{{end}}

<div class="table-responsive">
<table class="table table-striped table-sm">
              <thead>
                <tr>
                  <th>行业名</th>
                  <th>广告位所属行业{{if eq "32" (index .ARGS.entitytype_id 0)}}<br />
<input type=radio name=mychannel value="Inherit" {{if eq "Inherit" (index .ARGS.mychannel 0)}}checked{{end}} />Inherit
<input type=radio name=mychannel value="Own" {{if eq "Own" (index .ARGS.mychannel 0)}}checked{{end}} />Own{{end}}
</th>
                  <th>黑白名单设置<br />{{if eq "32" (index .ARGS.entitytype_id 0)}}
<input type=radio name=channel_order value="Inherit" {{if eq "Inherit" (index .ARGS.channel_order 0)}}checked{{end}} />系统默认{{end}}
<input type=radio name=channel_order value="Black" {{if eq "Black" (index .ARGS.channel_order 0)}}checked{{end}} />黑名单
<input type=radio name=channel_order value="White" {{if eq "White" (index .ARGS.channel_order 0)}}checked{{end}} />白名单
</th>
                </tr>
              </thead>
              <tbody>{{ with .Lists }}{{ range . }}
<tr><td>{{.channel_name}}</td>
<td><input name=belong_ids type=checkbox {{if .chbelong_id}}checked{{end}} value="{{.channel_id}}" /></td>
<td><input name=ac_ids type=checkbox {{if .chac_id}}checked{{end}} value="{{.channel_id}}" /></td>
</tr>{{end}}{{end}}
</tobdy>
</table>
</div>
<input type=submit value=" 保存 " />
</form>
{{ template "footer" }}
