{{ template "header" .}}
{{ template "chacheader" .}}

<form class="form" method=post action="chac">
<input type=hidden name=campaign_id value="{{index .ARGS.campaign_id 0}}" />
<input type=hidden name=campaign_md5 value="{{index .ARGS.campaign_md5 0}}" />
<input type=hidden name=campaign_name value="{{index .ARGS.campaign_name 0}}" />
<input type=hidden name=entitytype_id value="41" />
<input type=hidden name=action value="update" />

<h3>{{index .ARGS.campaign_name 0}}</h3>
<div class="table-responsive">
<table class="table table-striped table-sm">
              <thead>
                <tr>
                  <th>行业名</th>
                  <th>活动所属行业</th>
                  <th>黑白名单设置<br />
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
