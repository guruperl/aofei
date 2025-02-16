{{ template "header" .}}
{{ template "acheader" .}}

<form name=f1 class="form" method=post action="ac">
<input type=hidden name=action value="updateOrder" />
{{if eq "41" (index .ARGS.entitytype_id 0)}}
<input type=hidden name=campaign_id value="{{index .ARGS.campaign_id 0}}" />
<input type=hidden name=campaign_md5 value="{{index .ARGS.campaign_md5 0}}" />
<input type=hidden name=campaign_name value="{{index .ARGS.campaign_name 0}}" />
<input type=hidden name=entitytype_id value="41" />
<h3>{{index .ARGS.campaign_name 0}}</h3>{{else}}
<input type=hidden name=entitytype_id value="4" />
<h3>{{index .ARGS.a_company 0}}</h3>{{end}}

Access logic: <input type=radio name=access_order value="Black" {{if eq `Black` (index .ARGS.access_order 0)}}checked{{end}} />Black
<input type=radio name=access_order value="White" {{if eq `White` (index .ARGS.access_order 0)}}checked{{end}} />White
{{if eq `41` (index .ARGS.entitytype_id 0)}}<input type=radio name=access_order value="Inherit" {{if eq `Inherit` (index .ARGS.access_order 0)}}checked{{end}} />Inherit{{end}}
<input onClick="return (confirm('This will delete all existing access list and reset logic. Do you want to continue?')) ? true : false;" type=submit value=" Update " />
</form>

{{if ne `Inherit` (index .ARGS.access_order 0)}}


<div class="table-responsive">
<table class="table table-striped table-sm">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>URL</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>{{ with .Lists }}{{ range . }}
<tr><td>{{.company}}</td>
<td>{{.url}}</td>
<td><a href="ac?action=delete&ac_id={{.ac_id}}&{{if "41" (index .ARGS.entitytype_id 0)}}campaign_id={{index .ARGS.campaign_id 0}}&campaign_md5={{index .ARGS.campaign_md5 0}}&campaign_name={{index .ARGS.campaign_name 0 | urlquery}}&entitytype_id=41{{else}}entitytype_id=4{{end}}">Del</a></td>
</tr>{{end}}{{end}}
</tobdy>
</table>
</div>

<form name=f2 class="form" method=post action="ac">
<input type=hidden name=action value="insert" />
<input type=hidden name=othertype_id value="31" />
{{if eq "41" (index .ARGS.entitytype_id 0)}}
<input type=hidden name=campaign_id value="{{index .ARGS.campaign_id 0}}" />
<input type=hidden name=campaign_md5 value="{{index .ARGS.campaign_md5 0}}" />
<input type=hidden name=campaign_name value="{{index .ARGS.campaign_name 0}}" />
<input type=hidden name=entitytype_id value="41" />{{else}}
<input type=hidden name=entitytype_id value="4" />{{end}}
Add site by ID: <input type=text name=other_id size=12 />
<input type=submit value=" Add Site " />
</form>


{{end}}

{{ template "footer" }}
