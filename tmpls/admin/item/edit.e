<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
  <head>
    <title>Edit Item</title>
    <script src="../../../js/jquery-1.4.2.min.js"></script>
    [% INCLUDE form_ui_start.e %]
  </head>
<body>
  <script>
    $( document ).ready(
      function() {        
        $( "#btnSubmit" ).click(
          function() {
            var OUTCASTS = [ '!', '#', '$', '%', '^', '|', '{', '}' ]
            var RE_TIME = /\d{4}\-\d{2}\-\d{2}\s{1}\d{2}\:\d{2}\:\d{2}/
            var ERROR_GENERAL = "Invalid parameter value(s):\n"              
            var ERROR_STARTTIME = "Invalid start time.\n"
            var ERROR_ENDTIME = "Invalid end time.\n"
            var ERROR_QUANTITY = "Invalid quantity.\n"
            var ERROR_CPM_FC = "Invalid CPM FC.\n"      
            var ERROR_CPM_LENGTH = "Invalid CPM length.\n"          
            var ERROR_CPM_THROTTLE = "Invalid CPM throttle.\n"
            var ERROR_CPC_FC = "Invalid CPC FC.\n"      
            var ERROR_CPC_LENGTH = "Invalid CPC length.\n"
            var ERROR_CPA_FC = "Invalid CPA FC.\n"                    
            var ERROR_CPA_LENGTH = "Invalid CPA length.\n"
            var value = new String()
            var i = 0
            var string2int = null
            var invalids = new Array()
            var isIllegal = false
            var mve = new String()

            // check for invalid input
            $( ".Editable" ).each(
              function() {                  
                i = 0
                var count = 0              
                    
                for ( ; i < OUTCASTS.length; i++ ) {
                  if ( this.value.indexOf( OUTCASTS[ i ] ) > -1 ) {
                    invalids.push( this.id )
                    break                      
                  }                      
                }                  
              }
            )
              
            // check for invalid characters
            if ( invalids.length > 0 ) {
              mve = ERROR_GENERAL + invalids              
              $( "#message" )[ 0 ].value = mve
              return false
            }
              
            // check for valid start time
            value = $( "#startx" )[ 0 ].value
              
            if ( value.length > 0 ) {
              if ( ! RE_TIME.test( value ) ) {
                mve = ERROR_STARTTIME
                $( "#message" )[ 0 ].value = mve
                return false
              }
            }
              
            // check for valid end time
            value = $( "#endx" )[ 0 ].value
              
            if ( value.length > 0 ) {
              if ( ! RE_TIME.test( value ) ) {
                mve = ERROR_ENDTIME
                $( "#message" )[ 0 ].value = mve
                return false
              }
            }              
              
            // check for valid quantity
            value = $( "#quantity" )[ 0 ].value
              
            for ( ; i < value.length; i++ ) {
              string2int = parseInt( value[ i ] )

              if ( ( string2int >= 0 ) || ( string2int <= 9 ) ) {
                continue
              }
              else {
                isIllegal = true
                break                
              }
            }
                
            if ( isIllegal ) {
              mve += ERROR_QUANTITY
              $( "#message" )[ 0 ].value = mve
              return false                    
            }
              
            // check for valid cpm fc
            value = $( "#cpm_fc" )[ 0 ].value             
              
            for ( i = 0 ; i < value.length; i++ ) {
              string2int = parseInt( value[ i ] )

              if ( ( string2int >= 0 ) || ( string2int <= 9 ) ) {
                continue
              }
              else {
                isIllegal = true
                break                
              }
            }
                
            if ( isIllegal ) {
               mve += ERROR_CPM_FC
               $( "#message" )[ 0 ].value = mve
               return false                    
            }
                            
            // check for valid cpm length
            value = $( "#cpm_length" )[ 0 ].value             
              
            for ( i = 0 ; i < value.length; i++ ) {
              string2int = parseInt( value[ i ] )

              if ( ( string2int >= 0 ) || ( string2int <= 9 ) ) {
                continue
              }
              else {
                isIllegal = true
                break                
              }
            }
                
            if ( isIllegal ) {
              mve += ERROR_CPM_LENGTH
              $( "#message" )[ 0 ].value = mve
              return false                    
            }              
              
            // check for valid cpm throttle
            value = $( "#cpm_throttle" )[ 0 ].value             
              
            for ( i = 0 ; i < value.length; i++ ) {
              string2int = parseInt( value[ i ] )

              if ( ( string2int >= 0 ) || ( string2int <= 9 ) ) {
                continue
              }
              else {
                isIllegal = true
                break                
              }
            }
                
            if ( isIllegal ) {
              mve += ERROR_CPM_THROTTLE
              $( "#message" )[ 0 ].value = mve
              return false                    
            }              
              
            // check for valid cpc fc
            value = $( "#cpc_fc" )[ 0 ].value             
              
            for ( i = 0 ; i < value.length; i++ ) {
                  string2int = parseInt( value[ i ] )

              if ( ( string2int >= 0 ) || ( string2int <= 9 ) ) {
                continue
              }
              else {
                isIllegal = true
                break                
              }
            }
        
            if ( isIllegal ) {
              mve += ERROR_CPM_THROTTLE
              $( "#message" )[ 0 ].value = mve
              return false                    
            }
              
            // check for valid cpc length
            value = $( "#cpc_length" )[ 0 ].value             
              
            for ( i = 0 ; i < value.length; i++ ) {
                  string2int = parseInt( value[ i ] )

              if ( ( string2int >= 0 ) || ( string2int <= 9 ) ) {
                continue
              }
              else {
                isIllegal = true
                break                
              }
            }
        
            if ( isIllegal ) {
              mve += ERROR_CPC_LENGTH
              $( "#message" )[ 0 ].value = mve
              return false                    
            }
                        
            // check for valid cpa fc
            value = $( "#cpa_fc" )[ 0 ].value             
              
            for ( i = 0 ; i < value.length; i++ ) {
                  string2int = parseInt( value[ i ] )

              if ( ( string2int >= 0 ) || ( string2int <= 9 ) ) {
                continue
              }
              else {
                isIllegal = true
                break                
              }
            }
        
            if ( isIllegal ) {
              mve += ERROR_CPA_FC
              $( "#message" )[ 0 ].value = mve
              return false                    
            }
                        
            // check for valid cpa_length
            value = $( "#cpa_length" )[ 0 ].value             
              
            for ( i = 0 ; i < value.length; i++ ) {
                  string2int = parseInt( value[ i ] )

              if ( ( string2int >= 0 ) || ( string2int <= 9 ) ) {
                continue
              }
              else {
                isIllegal = true
                break                
              }
            }
        
            if ( isIllegal ) {
              mve += ERROR_CPA_LENGTH
              $( "#message" )[ 0 ].value = mve
              return false                    
            }            
          }
        ) 
      }  
    )  
  </script>
   <h2 align="center" class="curTitle" >Edit Item <a href="javascript:window.history.back()"><img src="/uilib/comImg/back.png" border=0 width="25" height="25" /></a></h2>
<div class="ui-layout-center">
 <div id="container">
    <div id="mainmenu">
					<ul id="tabs">
						<li>
							<a href="#EditItem">Edit Item</a>
						</li>
					</ul>
				<div>
				<div class="bar">&nbsp;</div>
				<div class="panel" id="EditItem">
[% SET item = edit.0 %]

<form class=jqtransform method=post action=item>
<input type="hidden" name="action" value="update" />
<input type="hidden" id="itemid" name="itemid" value="[% GET item.itemid %]" />
<input type="hidden" id="campaignid" name="campaignid" value="[% GET item.campaignid %]" />
<input type="hidden" name="campaignmd5" value="[% GET campaignmd5 %]" />
<input type="hidden" name="campaignname_esc" value="[% GET campaignname_esc %]" />

<table border=0>
<tr><td>Lineitem Name:</td><td><input type=text id="itemname" name="itemname" class="Editable" value='[% item.itemname %]' size=40></td></tr>
<tr>
  <td>
    Status:
  </td>  
  <td>
    <select id="status" name="status" multiple="true">
      <option value="Yes" [% IF item.status == 'Yes' %] selected="true" [% END %]>Yes</option>
      <option value="No" [% IF status.status %] selected="true" [% END %]>No</option>
      <option value="Pause" [% IF item.status == 'Pause' %] selected="true" [% END %]>Pause</option>
    </select>
  </td>
</tr>
<tr><td>Size:</td><td><select size=1 name=sizeid>
<option [% IF item.sizeid==1 %]selected[% END %] value=1>Half Banner 234x60</option>
<option [% IF item.sizeid==2 %]selected[% END %] value=2>Banner 468x60</option>
<option [% IF item.sizeid==3 %]selected[% END %] value=3>Leaderboard 728x90</option>
<option [% IF item.sizeid==4 %]selected[% END %] value=4>Micro Bar 88x31</option>
<option [% IF item.sizeid==5 %]selected[% END %] value=5>Button 120x60</option>
<option [% IF item.sizeid==6 %]selected[% END %] value=6>Button 120x90</option>
<option [% IF item.sizeid==7 %]selected[% END %] value=7>Button 125x125</option>
<option [% IF item.sizeid==8 %]selected[% END %] value=8>Vertical Banner 120x240</option>
<option [% IF item.sizeid==9 %]selected[% END %] value=9>Skyscraper 120x600</option>
<option [% IF item.sizeid==10 %]selected[% END %] value=10>Wide Skyscraper 160x600</option>
<option [% IF item.sizeid==11 %]selected[% END %] value=11>Vertical Rectangle 240x400</option>
<option [% IF item.sizeid==12 %]selected[% END %] value=12>Small Rectangle 180x150</option>
<option [% IF item.sizeid==13 %]selected[% END %] value=13>Small Square 200x200</option>
<option [% IF item.sizeid==14 %]selected[% END %] value=14>Square 250x250</option>
<option [% IF item.sizeid==15 %]selected[% END %] value=15>3:1 Rectangle 300x100</option>
<option [% IF item.sizeid==16 %]selected[% END %] value=16>Medium Rectangle 300x250</option>
<option [% IF item.sizeid==17 %]selected[% END %] value=17>Large Rectangle 336x280</option>
<option [% IF item.sizeid==18 %]selected[% END %] value=18>Half Page Ad 300x600</option>
</select></td></tr>
<tr><td>Cost Type:</td><td>
<input [% IF item.costtype=='CPD' %]checked[% END %] type=radio name=costtype value=CPD><label>CPD</label>
<input [% IF item.costtype=='CPM' %]checked[% END %] type=radio name=costtype value=CPM><label>CPM</label>
<input [% IF item.costtype=='CPC' %]checked[% END %] type=radio name=costtype value=CPC><label>CPC</label>
<input [% IF item.costtype=='CPA' %]checked[% END %] type=radio name=costtype value=CPA><label>CPA</label>
&nbsp;
<label> &nbsp; Price:</label> <input type=text id="cost" name="cost" class="Editable" value='[% item.cost %]' size=5>
</td></tr>
<tr><td>Start:</td><td><input type=text id="startx" name="startx" class="Editable" value='[% item.startx %]' size=20>
<label> &nbsp; End:</label> <input type=text id="endx" name="endx" class="Editable" value='[% item.endx %]' size=20></td></tr>
<tr><td>Quantity:</td><td><input type=text" id="quantity" name="quantity" class="Editable" value='[% item.quantity %]' size=9>
<label> &nbsp; Delivery Rate:</label> 
<select size=1 name=deliveryrate>
<option [% IF item.deliveryrate=='Fast' %]selected[% END %] value='Fast'>Fast</option>
<option [% IF item.deliveryrate=='Even' %]selected[% END %] value='Even'>Even</option>
</select></td></tr>
<tr><td valign=top>Frequency Caps:</td><td>
<table>
<tr><th>Type</th><th>Number</th><th>Period</th><th>Throttle</th></tr>
<tr><td>Impression</td>
<td><input type=text value='[% item.cpm_fc %]' id="cpm_fc" name="cpm_fc" class="Editable" size=3></td>
<td><input type=text value='[% item.cpm_length %]' id="cpm_length" name="cpm_length" class="Editable" size=9></td>
<td><input type=text value='[% item.cpm_throttle %]' id="cpm_throttle" name="cpm_throttle" class="Editable" size=9></td></tr>
<tr><td>Clicks</td>
<td><input type=text value='[% item.cpc_fc %]' id="cpc_fc" name="cpc_fc" class="Editable" size=3></td>
<td><input type=text value='[% item.cpc_length %]' id="cpc_length" name="cpc_length" class="Editable" size=9></td>
<td></td></tr>
<tr><td>Actions</td>
<td><input type=text value='[% item.cpa_fc %]' id="cpa_fc" name="cpa_fc" class="Editable" size=3></td>
<td><input type=text value='[% item.cpa_length %]' id="cpa_length" name="cpa_length" class="Editable" size=9></td>
<td></td></tr>
</table>
</td></tr>
[% INCLUDE edit_slot.e %]
<tr><td colspan=2> &nbsp; </td></tr>
</table>
      <br/>
      <div align="center">
        <input type="reset" value="Reset" />
        &nbsp;
        <input id="btnSubmit" type="submit" value="Update Lineitem" />
        <br/>
        <br/>
        <textarea id="message" cols="50" rows="2" style="display:none;"></textarea>
      </div>

</form>
   <div><!--<div class="panel" id="EditItem">-->
 <div><!--<div id="container">-->
[% INCLUDE form_ui_end.e %]