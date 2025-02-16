<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
  <head>
    <title>Edit Campaign</title>
    <script src="../../../js/jquery-1.4.2.min.js"></script>
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
            }
          ) 
        }  
      )  
    </script>
[% SET item=edit.0 %]
<h3>[% item.campaignname %]</h3>
<form class=jqtransform method=post action=campaign>
<input id="advid" name="advid" type="hidden" value="[% GET item.advid %]">
<input type=hidden name='campaignid' value='[% campaignid %]'>
<input type=hidden name='action' value='update'>
<table border="0">
<tr>
  <td>Campaign Name:</td>
  <td>
    <input type="text" id="campaignname" name="campaignname" class="Editable" value="[% item.campaignname %]" size="40" />
  </td>
</tr>
<tr>
  <td>Access Order:</td>
  <td>
    <select id="accessorder" name="accessorder" multiple="true">
      <option value="AllowDeny" [% IF item.accessorder == 'AllowDeny' %] selected="true" [% END %]>Allow, Deny</option>
      <option value="DenyAllow" [% IF item.accessorder == 'DenyAllow' %] selected="true" [% END %]>Deny, Allow</option>
      <option value="No" [% IF item.accessorder == 'No' %] selected="true" [% END %]>No</option>
    </select>
  </td>
</tr>
<tr>
  <td>Status:</td>
  <td>
    <select id="status" name="status" multiple="true">
      <option value="Yes" [% IF item.status == 'Yes' %] selected="true" [% END %]>Yes</option>
      <option value="No" [% IF item.status == 'No' %] selected="true" [% END %]>No</option>
      <option value="Pause" [% IF item.status == 'Pause' %] selected="true" [% END %]>Pause</option>
    </select>
  </td>
</tr>
<tr><td>Foreign ID: </td><td><input type=text id="foreignid" name="foreignid" class="Editable" value='[% item.foreignid %]' size=10></td></tr>
<tr><td>Start:</td><td><input type=text id="startx" name="startx" class="Editable" value='[% item.startx %]' size=16>
<label> &nbsp; End:</label> <input type=text id="endx" name="endx" class="Editable" value='[% item.endx %]' size=16></td></tr>
<tr><td valign=top>Frequency Caps:</td><td>
<table>
<tr><th>Type</th><th>Number</th><th>Period</th><th>Throttle</th></tr>
<tr><td>Impression</td>
<td><input type=text value='[% item.cpm_fc %]' id="cpm_fc" name="cpm_fc" size=3  class="Editable"></td>
<td><input type=text value='[% item.cpm_length %]' id=="cpm_length" name="cpm_length" class="Editable" size=9></td>
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
[% INCLUDE edit_site.e %]
[% IF GOTOINSTALL=='adv' %][% INCLUDE edit_pub.e %][% END %]
[% IF GOTOINSTALL=='pub' %][% INCLUDE edit_adv.e %][% END %]
<tr><td colspan=2> &nbsp; </td></tr>
</table>
      <br/>
      <div align="center">
        <input type="reset" value="Reset" />
        &nbsp;
        <input id="btnSubmit" type="submit" value="Update" />      
        <br/>
        <br/>
        <textarea id="message" cols="50" rows="2"></textarea>
      </div>
</form>

</div>

  </body>
</html>
