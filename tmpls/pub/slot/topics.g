{{$attach := print "site_id=" (index .ARGS.site_id 0) "&site_md5=" (index .ARGS.site_md5 0) "&site_name=" (index .ARGS.site_name 0 | urlquery)}}
{{ template "header" .}}
{{ template "slotheader" .}}

<h2>广告位列表</h2>
<div class="table-responsive">
<table class="table table-striped table-sm">
              <thead>
                <tr>
                  <th>广告位名称</th>
                  <th></th>
                  <th></th>
                  <th></th>
                  <th></th>
                  <th></th>
                </tr>
              </thead>
              <tbody>{{ range .Lists }}
<tr><td>{{.slot_name}}</td>
{{$small := print "slot_id=" .slot_id "&slot_md5=" .slot_md5 "&slot_name=" (.slot_name | urlquery)}}
<td><a href="/pz/{{.site_id}}/{{.slot_id}}.js">Ads</a></td>
<td><a href="chac?action=topics&entitytype_id=32&{{$small}}&{{$attach}}">行业设置</a></td>
<td><a href="slot?action=edit&slot_id={{.slot_id}}&{{$attach}}">编辑</a></td>
<td><a onClick="return (confirm('Do you want to remove your site {{.slot_name}}?')) ? true : false;" href="slot?action=delete&slot_id={{.slot_id}}&{{$attach}}">删除</a></td>
{{end}}</tobdy>

</table>
</div>

{{ template "footer" }}
