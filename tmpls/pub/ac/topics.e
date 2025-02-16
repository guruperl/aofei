{{ template "header" .}}
{{ template "acheader" .}}

<form name=f1 class="form" method=post action="ac">
<input type=hidden name=action value="updateOrder" />
{{if eq "31" (index .ARGS.entitytype_id 0)}}
<input type=hidden name=site_id value="{{index .ARGS.site_id 0}}" />
<input type=hidden name=site_md5 value="{{index .ARGS.site_md5 0}}" />
<input type=hidden name=site_name value="{{index .ARGS.site_name 0}}" />
<input type=hidden name=entitytype_id value="31" />
<h3>{{index .ARGS.site_name 0}}</h3>{{else}}
<input type=hidden name=entitytype_id value="3" />
<h3>{{index .ARGS.p_company 0}}</h3>{{end}}

Access logic: <input type=radio name=access_order value="Black" {{if eq `Black` (index .ARGS.access_order 0)}}checked{{end}} />Black
<input type=radio name=access_order value="White" {{if eq `White` (index .ARGS.access_order 0)}}checked{{end}} />White
{{if eq `31` (index .ARGS.entitytype_id 0)}}<input type=radio name=access_order value="Inherit" {{if eq `Inherit` (index .ARGS.access_order 0)}}checked{{end}} />Inherit{{end}}
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
<td><a href="ac?action=delete&ac_id={{.ac_id}}&{{if "31" (index .ARGS.entitytype_id 0)}}site_id={{index .ARGS.site_id 0}}&site_md5={{index .ARGS.site_md5 0}}&site_name={{index .ARGS.site_name 0 | urlquery}}&entitytype_id=31{{else}}entitytype_id=3{{end}}">Del</a></td>
</tr>{{end}}{{end}}
</tobdy>
</table>
</div>

<form name=f2 class="form" method=post action="ac">
<input type=hidden name=action value="insert" />
<input type=hidden name=othertype_id value="4" />
{{if eq "31" (index .ARGS.entitytype_id 0)}}
<input type=hidden name=site_id value="{{index .ARGS.site_id 0}}" />
<input type=hidden name=site_md5 value="{{index .ARGS.site_md5 0}}" />
<input type=hidden name=site_name value="{{index .ARGS.site_name 0}}" />
<input type=hidden name=entitytype_id value="31" />{{else}}
<input type=hidden name=entitytype_id value="3" />{{end}}
Add advertiser by ID: <input type=text name=other_id size=12 />
<input type=submit value=" Add Advertiser " />
</form>


{{end}}

{{ template "footer" }}
