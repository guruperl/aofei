{{ template "header" .}}
{{ template "siteheader" .}}

<h3>广告位组</h3>
<div class="table-responsive">
<table class="table table-striped table-sm">
              <thead>
                <tr>
                  <th>广告组名称</th>
                  <th>广告组URL</th>
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
<td><a href="slot?action=topics&site_id={{.site_id}}&site_md5={{.site_md5}}&site_name={{.site_name | urlquery }}">广告位</a></td>
<td><a href="chac?action=topics&site_id={{.site_id}}&site_md5={{.site_md5}}&site_name={{.site_name | urlquery }}&entitytype_id=31">行业设置</a></td>
<td><a href="ac?action=topics&site_id={{.site_id}}&site_md5={{.site_md5}}&site_name={{.site_name | urlquery }}&entitytype_id=31">网站黑白名单</a></td>
<td><a href="site?action=edit&site_id={{.site_id}}">编辑</a></td>
<td><a onClick="return (confirm('Do you want to remove your site {{.site_name}}?')) ? true : false;" href="site?action=delete&site_id={{.site_id}}">删除</a></td>
</tr>
{{end}}{{end}}</tobdy>
</table>
</div>

{{ template "footer" }}
