{{$attach := print "campaign_id=" (index .ARGS.campaign_id 0) "&campaign_md5=" (index .ARGS.campaign_md5 0) "&campaign_name=" (index .ARGS.campaign_name 0 | urlquery)}}

{{ template "header" .}}
{{ template "itemheader" .}}

<h3>{{index .ARGS.campaign_name 0}}</h3>

<div class="table-responsive">
<table class="table table-striped table-sm">
<thead><tr>
<th>Name</th>
<th>Price</th>
<th>Placing</th>
<th>Start</th>
<th>End</th>
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
<td><a href="balance?action=topics&{{$attach}}&{{$second}}&entitytype_id=42">Finance</a></td>
<td><a href="creative?action=topics&{{$attach}}&{{$second}}">Creatives</a></td>
<td><a href="item?action=edit&{{$attach}}&{{$second}}">Edit</a></td>
<td><a href="item?action=delete&{{$attach}}&{{$second}}">Del</a></td>
</tr>{{end}}{{end}}
</tbody>
</table>
</div>

{{template "footer"}}
