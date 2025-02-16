{{$attach := print "campaign_id=" (index .ARGS.campaign_id 0) "&campaign_md5=" (index .ARGS.campaign_md5 0) "&campaign_name=" (index .ARGS.campaign_name 0 | urlquery)}}

{{ template "header" .}}
{{ template "itemheader" .}}

<h3>{{index .ARGS.campaign_name 0}}</h3>

<div class="table-responsive">
<table class="table table-striped table-sm">
<thead><tr>
<th>创意名</th>
<th>价格</th>
<th>投放平台</th>
<th>投放起始时间</th>
<th>投放终止时间</th>
<th></th>
<th></th>
<th></th>
<th></th>
</tr></thead>
<tbody>{{with .Lists}}{{range .}}
<td>{{.item_name}}</td>
<td>{{.cost}} {{.cost_type}}</td>
<td>{{.fl_platform}}</td>
<td>{{.startx}}</td>
<td>{{.endx}}</td>
{{$second := print "item_id=" .item_id "&item_md5=" .item_md5 "&item_name=" (.item_name | urlquery)}}
<td><a href="balance?action=topics&{{$attach}}&{{$second}}&entitytype_id=42">花费</a></td>
<td><a href="creative?action=topics&{{$attach}}&{{$second}}">创意素材</a></td>
<td><a href="item?action=edit&{{$attach}}&{{$second}}">编辑</a></td>
<td><a href="item?action=delete&{{$attach}}&{{$second}}">删除</a></td>
</tr>{{end}}{{end}}
</tbody>
</table>
</div>

{{template "footer"}}
