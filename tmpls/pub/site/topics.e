{{ template "header" .}}
{{ template "siteheader" .}}

<h3>Current Sites</h3>
<div class="table-responsive">
<table class="table table-striped table-sm">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>URL</th>
                  <th></th>
                  <th></th>
                  <th></th>
                  <th></th>
                  <th></th>
                </tr>
              </thead>
              <tbody>{{ with .Lists }}{{ range . }}
<tr>
<td>{{.site_name}}</td><td>{{.site_url}}</td>
<td><a href="slot?action=topics&site_id={{.site_id}}&site_md5={{.site_md5}}&site_name={{.site_name | urlquery }}">Slots</a></td>
<td><a href="chac?action=topics&site_id={{.site_id}}&site_md5={{.site_md5}}&site_name={{.site_name | urlquery }}&entitytype_id=31">Channels</a></td>
<td><a href="ac?action=topics&site_id={{.site_id}}&site_md5={{.site_md5}}&site_name={{.site_name | urlquery }}&entitytype_id=31">BW</a></td>
<td><a href="site?action=edit&site_id={{.site_id}}">Edit</a></td>
<td><a onClick="return (confirm('Do you want to remove your site {{.site_name}}?')) ? true : false;" href="site?action=delete&site_id={{.site_id}}">Del</a></td>
</tr>
{{end}}{{end}}</tobdy>
</table>
</div>

{{ template "footer" }}
