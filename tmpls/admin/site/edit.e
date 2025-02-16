<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
  <head>
    <title>Edit Site</title>
    <script src="../../../js/jquery-1.4.2.min.js"></script>
    [% INCLUDE form_ui_start.e %]
    <style type="text/css">
    table td{
    	vertical-align:top;
    }
    </style>
  </head>
<body>
  <script>
    // nicer implementation of Array.toString()
    function array2string( arr ) {
      var i = 0
      var a2s = new String()     
  
      if ( arr == null ) {
        return null
      }
      else {
        if ( arr.length > 0 ) {
          for ( ; i < arr.length; i++ ) {
            if ( i == 0 ) {
              a2s += arr[ i ] + '.'
            }
            else if ( i > 0 ) {
              if ( ( arr.length - i ) > 1 ) {
                a2s += arr[ i ] + ", "
              }            
              else {
                a2s += arr[ i ] + '.'
              }
            }  
          }
        }
          
        return a2s            
      }
    }
        
    $( document ).ready(
      function() {        
        $( "#btnSubmit" ).click(
          function() {
            var OUTCASTS = [ '!', '#', '$', '%', '^', '|', '{', '}' ]
            var RE_URL = /.*\.[a-z]{2,4}$/
            var ERROR_CHARACTERS = "Invalid characters in these fields: "
            var ERROR_URL = "\nInvalid URL.\n"              
            var invalids = new Array()
            var i = 0
            var count = 0
            var messages = new Array()

            // clean up message board
            $( "#message" )[ 0 ].value = new String()
            
            // check for invalid input
            $( ".Editable" ).each(
              function() {                    
                for ( i = 0 ; i < OUTCASTS.length; i++ ) {
                  if ( this.value.indexOf( OUTCASTS[ i ] ) > -1 ) {
                    invalids.push( this.id )
                    break                      
                  }                      
                }                  
              }
            )
              
            // check for invalid characters
            if ( invalids.length > 0 ) {
              messageContent = ERROR_CHARACTERS + invalids
              messages.push( messageContent )
            }  
              
            // check URL
            value = $( "#siteurl" )[ 0 ].value
              
            for ( i = 0; i < value.length; i++ ) {
              if ( value[ i ].indexOf( '.' ) == -1 ) {
                count++                                  
              }                      
            }

            if ( count == value.length || ! RE_URL.test( value ) ) {
              messages.push( ERROR_URL )
            }            
            
            // display error messages & stop form action
            if ( messages.length > 0 ) {
              $( "#messageBoard" )[ 0 ].innerText = array2string( messages )
              return false
            }            
          }
        ) 
      }  
    )  
  </script>
[% SET item=edit.0 %]
<h2 align="center" class="curTitle" >[% item.sitename %] <a href="javascript:window.history.back()"><img src="/uilib/comImg/back.png" border=0 width="25" height="25" /></a></h2>
<form class=jqtransform method=post action=site>
<div id="container">
    <div id="mainmenu">
					<ul id="tabs">
						<li>
							<a href="#EditPublisher">Edit Site([% item.sitename %])</a>
						</li>
					</ul>
				<div>
				<div class="bar">&nbsp;</div>
				<div class="panel" id="EditPublisher">
<input id="action" name="action" type="hidden" value="update" />
<input id="pubid" name="pubid" type="hidden" value=[% GET item.pubid %] />
<input type=hidden name='siteid' value='[% item.siteid %]'>
<table border=0>
<tr>
  <td>Site Name:</td>
  <td>
    <input type="text" value="[% item.sitename %]" id="sitename" name="sitename" class="Editable" size="40" />
  </td>
</tr>
<table>
  <tr>
<!--  
    <td>Priority:</td>
    <td>
      <select id="priority" name="priority" multiple="true">
        <option [% IF item.priority == 'High' %] selected="true" [% END %] value="High">High</option>
        <option [% IF item.priority == 'Standard' %] selected="true" [% END %] value="Standard">Standard</option>
        <option [% IF item.priority == 'Low' %] selected="true" [% END %] value="Low">Low</option>
      </select>
    </td>
-->    
    <td>Channel&nbsp;Order:</td>
    <td>
      <select id="channelorder" name="channelorder" multiple="true">
        <option [% IF item.channelorder == 'AllowDeny' %] selected="true" [% END %] value="AllowDeny">AllowDeny</option>
        <option [% IF item.channelorder == 'DenyAllow' %] selected="true" [% END %] value="DenyAllow">DenyAllow</option>      
        <option [% IF item.channelorder == 'No' %] selected="true" [% END %] value="No">No</option>
      </select>
    </td>
    <td>Access&nbsp;Order:</td>
    <td>
      <select id="accessorder" name="accessorder" multiple="true">
        <option [% IF item.accessorder == 'AllowDeny' %] selected="true" [% END %] value="AllowDeny">AllowDeny</option>
        <option [% IF item.accessorder == 'DenyAllow' %] selected="true" [% END %] value="DenyAllow">DenyAllow</option>      
        <option [% IF item.accessorder == 'No' %] selected="true" [% END %] value="No">No</option>
      </select>
    </td>
    <td>Status:</td>
    <td>
      <select id="status" name="status" multiple="true">
        <option [% IF item.status == 'Yes' %] selected="true" [% END %] value="Yes">Yes</option>
        <option [% IF item.status == 'No' %] selected="true" [% END %] value="No">No</option>
        <option [% IF item.status == 'New' %] selected="true" [% END %] value="New">New</option>
      </select>
    </td>
  </tr>
</table>
<tr><td>URL:</td><td><input type=text value='[% item.siteurl %]' id="siteurl" name="siteurl" class="Editable" size=40></td></tr>
<tr><td>Priority:</td><td>
<input type=radio [% IF item.priority=='High' %]checked[% END %] name=priority value="High"><label>High</label>
<input type=radio [% IF item.priority=='Standard' %]checked[% END %] name=priority value="Standard"><label>Standard</label>
<input type=radio [% IF item.priority=='Low' %]checked[% END %] name=priority value="Low"><label>Low</label>
</td></tr>
<tr><td colspan=2> <h3>Site Property</h3> </td></tr>
<tr><td>Language:</td><td><select size=1 name=languageid>
<option [% IF item.languageid==0 %]selected[% END %] value=0>Adjusted</option>
<option [% IF item.languageid==2 %]selected[% END %] value=2>Arabic</option>
<option [% IF item.languageid==3 %]selected[% END %] value=3>Chinese</option>
<option [% IF item.languageid==1 %]selected[% END %] value=1>English</option>
<option [% IF item.languageid==4 %]selected[% END %] value=4>French</option>
<option [% IF item.languageid==5 %]selected[% END %] value=5>Russian</option>
<option [% IF item.languageid==6 %]selected[% END %] value=6>Spanish</option>
</select></td></tr>
<tr><td>Reader Group:</td><td>
<input type=checkbox [% IF item.sp_r_men==1 %]checked[% END %] name=sp_reader value="Men" checked><label>Men</label>
<input type=checkbox [% IF item.sp_r_women==1 %]checked[% END %] name=sp_reader value="Women" checked><label>Women</label>
<input type=checkbox [% IF item.sp_r_teens==1 %]checked[% END %] name=sp_reader value="Teens"><label>Teens</label>
<input type=checkbox [% IF item.sp_r_kids==1 %]checked[% END %] name=sp_reader value="Kids"><label>Kids</label>
</td></tr>
<tr><td>Platform:</td><td>
<input type=radio [% IF item.sp_platform=="Web" %]checked[% END %] name=sp_platform value="Web" checked><label>Web</label>
<input type=radio [% IF item.sp_platform=="Mobile" %]checked[% END %] name=sp_platform value="Mobile"><label>Mobile</label>
<input type=radio [% IF item.sp_platform=="Email" %]checked[% END %] name=sp_platform value="Email"><label>Email</label>
<input type=radio [% IF item.sp_platform=="Video" %]checked[% END %] name=sp_platform value="Video"><label>Video</label>
<input type=radio [% IF item.sp_platform=="Device" %]checked[% END %] name=sp_platform value="Device"><label>Device</label>
</td></tr>
<tr><td>Site Type:</td><td>
<input type=radio name=sp_style [% IF item.sp_style=="Content" %]checked[% END %] value="Content"><label>Content</label>
<input type=radio name=sp_style [% IF item.sp_style=="Blog" %]checked[% END %] value="Blog"><label>Blog</label>
<input type=radio name=sp_style [% IF item.sp_style=="Directory" %]checked[% END %] value="Directory"><label>Directory</label>
<input type=radio name=sp_style [% IF item.sp_style=="Ecommerce" %]checked[% END %] value="Ecommerce"><label>Ecommerce</label>
<input type=radio name=sp_style [% IF item.sp_style=="Social" %]checked[% END %] value="Social"><label>Social Network</label>
<input type=radio name=sp_style [% IF item.sp_style=="Search" %]checked[% END %] value="Search"><label>Search</label>
</td></tr>
<tr><td>Sale Type:</td><td>
<input type=radio name=sp_vertical [% IF item.sp_vertical=="Direct" %]checked[% END %] value="Direct"><label>Direct</label>
<input type=radio name=sp_vertical [% IF item.sp_vertical=="Indirect" %]checked[% END %] value="Indirect"><label>Indirect</label>
</td></tr>
[% IF GOTOINSTALL=='adv' %][% INCLUDE edit_pub.e %][% END %]
[% IF GOTOINSTALL=='pub' %][% INCLUDE edit_adv.e %][% END %]
<tr><td colspan=2> &nbsp; </td><td>
</table>
      <br/>
      <div align="center">
        <input type="reset" value="Reset" />
        &nbsp;
        <input id="btnSubmit" type="submit" value="Update Site" />
        <br/>
        <br/>
        <textarea id="messageBoard" cols="50" rows="2" style="display:none;"></textarea>
              </div><!--<div class="panel" id="EditPublisher">-->
      </div><!--<div id="container">-->
      </div>
</form>

</div>
[% INCLUDE form_ui_end.e %]
[% INCLUDE end.e %]
