{{$attach := print "campaign_id=" (index .ARGS.campaign_id 0) "&campaign_md5=" (index .ARGS.campaign_md5 0) "&campaign_name=" (index .ARGS.campaign_name 0 | urlquery)}}
{{$second := print "item_id=" (index .ARGS.item_id 0) "&item_md5=" (index .ARGS.item_md5 0) "&item_name=" (index .ARGS.item_name 0 | urlquery)}}

{{ template "header" .}}
{{ template "creativeheader" .}}

<h3>{{index .ARGS.item_name 0}}</h3>

<form class="form" method=post action="creative">
<input type=hidden name=action value="insert" />
<input type=hidden name=campaign_id value="{{index .ARGS.campaign_id 0}}" />
<input type=hidden name=campaign_md5 value="{{index .ARGS.campaign_md5 0}}" />
<input type=hidden name=campaign_name value="{{index .ARGS.campaign_name 0}}" />
<input type=hidden name=item_id value="{{index .ARGS.item_id 0}}" />
<input type=hidden name=item_md5 value="{{index .ARGS.item_md5 0}}" />
<input type=hidden name=item_name value="{{index .ARGS.item_name 0}}" />

<div class="table-responsive">
<table class="table table-striped table-sm">
<thead><tr>
<th>Name</th>
<th>Weight</th>
<th>Content</th>
<th></th>
</tr></thead>
<tbody>{{with .Lists}}{{range .}}
<td>{{.creative_name}}</td>
<td>{{.weight}}</td>
<td>{{.content}}</td>
<td><a href="creative?action=delete&creative_id={{.creative_id}}&{{$attach}}&{{$second}}">Delete</a></td>
</tr>{{end}}{{end}}
<tr>
<td><input type=text name="creative_name" /></td>
<td><input type=text name="weight" size=5 /></td>
<td><input type=text name="content" /></td>
<td><input type=submit value="Add" /></td>
</tr>
</tbody>
</table>
</div>

{{template "footer"}}
