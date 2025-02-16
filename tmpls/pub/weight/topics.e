[% INCLUDE start.e %]
<div class="ui-layout-west">
    <ul>
        <li><a href="site?action=edit&siteid=[% siteid %]">[% sitename %]</a>
        <p></p>
		<ul><li><a href="slot?action=edit&slotid=[% slotid %]&siteid=[% siteid %]&sitemd5=[% sitemd5 %]&sitename_esc=[% sitename_esc %]">[% slotname %]</a>
			<p></p>
			<ul>
                <li>Trafficking Control</li>
           	</ul></li>
        </ul></li>
    </ul>
</div>


<div class="ui-layout-center"> 

<h2>Trafficking Control for [% slotname %]</h2>
<form name=weight method=post action=weight>
<input type=hidden name=action value='mulinsert'>
<input type=hidden name=slotid value='[% slotid %]'>
<input type=hidden name=slotmd5 value='[% slotmd5 %]'>
<input type=hidden name=slotname value='[% slotname %]'>
<input type=hidden name=siteid value='[% siteid %]'>
<input type=hidden name=sitemd5 value='[% sitemd5 %]'>
<input type=hidden name=sitename value='[% sitename %]'>

<table id=trafficktable border=0>
<tr>
<th>Item</th>
<th>Campaign</th>
<th>Start</th>
<th>End</th>
<th>Price</th>
<th>Priority</th>
<th>Scale</th>
<th><img src="/uilib/comImg/delete.gif" id="weightremove"></th>
</tr>
</table>

<p><p>

<input type=submit value="Update">
<p><p>
</form>

<select name="itemsToChoose" id="left" size="8" multiple="multiple">
</select>
<input id="leftbutton" value="add" type="button">


<script type="text/javascript">
var ITEMS = new Array();
var ins = new Array();
ins = [[% FOREACH item IN topics %]
{[% FOREACH pair IN item.pairs %]"[% pair.key %]" : "[% pair.value %]", [% END %]}, [% END %]
];
</script>
<script type="text/javascript" src="/cache/pub/[% pubid %]/weight/[% slotid %]_js.js?slotid=[% slotid %]&slotmd5=[% slotmd5 %]"></script>
<script type="text/javascript">
var outs = new Array();
for (var i=0; i<ITEMS.length; i++) {
  var found = false;
  for (var j=0; j<ins.length; j++) {
    if (ins[j]['itemid'] == ITEMS[i]['itemid']) {
      ins[j]['itemname']     = ITEMS[i]['itemname'];
      ins[j]['campaignname'] = ITEMS[i]['campaignname'];
      ins[j]['startx']       = ITEMS[i]['startx'];
      ins[j]['endx']         = ITEMS[i]['endx'];
      found = true;
      break;
    }
  }
  if (found==false) outs.push(ITEMS[i]);
}

for (var i=0; i<ins.length; i++)
	$("#trafficktable").append(trstr(ins[i]));

for (var i=0; i<outs.length; i++)
	$("#left").append(new Option(outs[i]["itemname"], outs[i]["itemid"], false, false));

function trstr(a) {
	return "<tr><td>"+a['itemname']+"</td><td>"+a['campaignname']+"</td><td><small>"+a['startx']+"</small></td><td><small>"+a['endx']+"</small></td><td>"+a['costtype']+":"+a['cost']+"</td><td><select size=1 name=priority"+a['itemid']+"> <option "+((a['priority']==0)?"selected":"")+" value=0>House</option> <option "+((a['priority']==1)?"selected":"")+" value=1>Low Network</option> <option "+((a['priority']==2)?"selected":"")+" value=2>High Network</option> <option "+((a['priority']==3)?"selected":"")+" value=3>RTB</option> <option "+((a['priority']==4)?"selected":"")+" value=4>Non-Guaranteed</option> <option "+((a['priority']==5)?"selected":"")+" value=5>Premium</option> <option "+((a['priority']==6)?"selected":"")+" value=6>Exclusive</option> </select></td><td><input type=text name=scale"+a['itemid']+" size=2 maxlength=2 value='"+((a['scale']==undefined)?'50':a['scale'])+"'><td><input type=radio name=itemid value='"+a['itemid']+"'></td></tr>";
}

function item_del(itemid) {
  var dest = new Array();
  for (var i=0; i<ins.length; i++) {
    if (ins[i]['itemid']==itemid) {
      outs.push(ins[i]);
    } else {
      dest.push(ins[i]);
    }
  }
  return dest;
}

function item_add(itemid) {
  var dest = new Array();
  for (var i=0; i<outs.length; i++) {
    if (outs[i]['itemid']==itemid) {
      ins.push(outs[i]);
    } else {
      dest.push(outs[i]);
    }
  }
  return dest;
}

$(function() {
  $("#weightremove").click(function(){
    var obj = $("input:radio[name='itemid']:checked");
    if (obj == undefined) return false;
    ins = item_del(obj.val());
    obj.closest('tr').remove();
    $("#left").append(new Option(outs[outs.length-1]["itemname"], outs[outs.length-1]["itemid"], false, false));
  });
  $("#leftbutton").click(function(){
    $("#left option:selected").each(function(){
      outs = item_add($(this).val());
      $("#trafficktable").append(trstr(ins[ins.length-1]));        
      $(this).remove();
    });
  });
});

</script>



</div>

[% INCLUDE end.e %]
