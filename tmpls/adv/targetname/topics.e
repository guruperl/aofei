{{template "header" .}}
{{template "targetnameheader" .}}

<style>
.tab-pane {

    border-left: 1px solid #ddd;
    border-right: 1px solid #ddd;
    border-bottom: 1px solid #ddd;
    border-radius: 0px 0px 5px 5px;
    padding: 10px;
}

.nav-tabs {
    margin-bottom: 0;
}
#exTab2 h3 {
  color : white;
  background-color: #428bca;
  padding : 5px 15px;
}
#exTab2 ul li {
  padding : 5px 5px;
}
</style>

<h3>{{index .ARGS.campaign_name 0}}</h3>
<form class="form" method=post action=targetname>
<input type=hidden name=campaign_id value="{{index .ARGS.campaign_id 0}}" />
<input type=hidden name=campaign_md5 value="{{index .ARGS.campaign_md5 0}}" />
<input type=hidden name=campaign_name value="{{index .ARGS.campaign_name 0}}" />
<input type=hidden name=action value="insert" />

<div id="exTab2" class="container">	
	<ul class="nav nav-tabs">
		<li class="nav-item active"><a href="#1" data-toggle="tab">Geography</a>
		</li>
		<li class="nav-item"><a href="#2" data-toggle="tab">Browser</a>
		</li>
		<li class="nav-item"><a href="#3" data-toggle="tab">Demography</a>
		</li>
		<li class="nav-item"><a href="#4" data-toggle="tab">Custom</a>
		</li>
		<li class="nav-item"><a href="#5" data-toggle="tab">Time</a>
		</li>
	</ul>

	<div class="tab-content">
		<div class="tab-pane active" id="1">
			<h3>States</h3>
<select name=state multiple>{{range $k, $v := .Other.state}}
<option {{if index $v 1}}selected{{end }} value="{{$k}}">{{index $v 0}}</option>{{end}}
</select>

			<h3>Cities</h3>
{{range $key, $val := .Other.city}}<h4>{{$key}}</h4>
<select name=city multiple>{{range $k, $v := $val}}
<option {{if index $v 1}}selected{{end }} value="{{$k}}">{{index $v 0}}</option>{{end}}
</select>
{{end}}
		</div>
		<div class="tab-pane" id="2">
			<h3>Browser</h3>
{{range $key, $val := .Other.pzua}}<h4>{{$key}}</h4>
<select name={{$key}} multiple>{{range $k, $v := $val}}
<option {{if index $v 1}}selected{{end }} value="{{$k}}">{{index $v 0}}</option>{{end}}
</select>
{{end}}
		</div>
		<div class="tab-pane" id="3">
			<h3>Demography</h3>
{{range $key, $val := .Other.demo}}<h4>{{$key}}</h4>
<select name="{{$key}}" multiple>{{range $k, $v := $val}}
<option {{if index $v 1}}selected{{end }} value="{{$k}}">{{index $v 0}}</option>{{end}}
</select>
{{end}}
		</div>
		<div class="tab-pane" id="4">
			<h3>Customized</h3>
{{range $key, $val := .Other.custom}}<h4>{{$key}}</h4>
<select name="{{$key}}"  multiple>{{range $k, $v := $val}}
<option {{if index $v 1}}selected{{end }} value="{{$k}}">{{index $v 0}}</option>{{end}}
</select>
{{end}}
		</div>
		<div class="tab-pane" id="5">
			<h3>Weekdays</h3>
<select name="weekday" multiple>{{range $k, $v:= .Other.weekday}}
<option {{if index $v 1}}selected{{end }} value="{{$k}}">{{index $v 0}}</option>{{end}}
</select>
			<h3>Hours</h3>
<select name="weekhour" multiple>{{range $k, $v:= .Other.weekhour}}
<option {{if index $v 1}}selected{{end }} value="{{$k}}">{{index $v 0}}</option>{{end}}
</select>
		</div>
	</div>
</div>

<input type=submit value=" Submit " />
</form>

{{template "footer"}}
