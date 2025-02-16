<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
  <head>
    <title>Edit Publisher</title>
    <script src="../../../js/jquery-1.4.2.min.js"></script>
    [% INCLUDE form_ui_start.e %]
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
              var RE_EMAIL = /\w+\@.*\w+/
              var RE_COUNTRY = /[A-Za-z]/
              var RE_IP = /[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}/
              var ERROR_CHARACTERS = "Invalid characters in these fields: "
              var ERROR_URL = "\nInvalid URL.\n"
              var ERROR_COUNTRY = "Invalid country.\n"
              var ERROR_EMAIL = "\nInvalid e-mail address.\n"
              var ERROR_PHONE = "\nInvalid phone number.\n"
              var ERROR_FAX = "\nInvalid fax number.\n"
              var ERROR_IP_FORMAT = "\nInvalid IP address format.\n"
              var ERROR_IP_VALUE = "\nInvalid IP address value.\n"              
              var value = new String()
              var i = 0
              var count = 0
              var string2int = 0
              var invalids = new Array()
              var isIllegal = false
              var messages = new Array()
              var messageContent = new String()
            
              // check for invalid input
              $( ".Editable" ).each(
                function() {                                      
                  for ( i = 0; i < OUTCASTS.length; i++ ) {
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
              value = $( "#url" )[ 0 ].value
              
              for ( i = 0; i < value.length; i++ ) {
                if ( value[ i ].indexOf( '.' ) == -1 ) {
                  count++                  
                }                      
              }
              
              if ( count == value.length || ! RE_URL.test( value ) ) {
                messages.push( ERROR_URL )
              }
              
              // check country
              value = $( "#country" )[ 0 ].value
              
              for ( i = 0; i < value.length; i++ ) { 
                if ( RE_COUNTRY.test( value[ i ] ) ) {
                  continue
                }
                else {
                  isIllegal = true                
                  break
                }  
              }              
              
              if ( isIllegal ) {
                messages.push( ERROR_COUNTRY )
              }              
              
              // check contact e-mail             
              if ( ! RE_EMAIL.test( $( "#contactemail" )[ 0 ].value ) ) {
                messages.push( ERROR_EMAIL )
              }
              
              // check phone
              value = $( "#phone" )[ 0 ].value
              isIllegal = false
              
              for ( i = 0; i < value.length; i++ ) { 
                string2int = parseInt( value[ i ] )

                if ( string2int >= 0 || string2int <= 9 ) {
                  continue
                }
                else {
                  isIllegal = true                
                  break
                }  
              }              
              
              if ( isIllegal ) {
                messages.push( ERROR_PHONE )                  
              }
              
              // check fax
              value = $( "#fax" )[ 0 ].value
              isIllegal = false
              
              for ( i = 0; i < value.length; i++ ) {
                string2int = parseInt( value[ i ] )

                if ( string2int >= 0 || string2int <= 9 ) {
                  continue
                }
                else {
                  isIllegal = true                
                  break
                }  
              }              
              
              if ( isIllegal ) {
                messages.push( ERROR_FAX )
              }                            

              // check IP address
              value = $( "#ip" )[ 0 ].value
                            
              if ( RE_IP.test( value ) ) {
                var ipparts = value.split( '.' )                
                isIllegal = false
                
                // check for illegal IP node & subnetwork values - lesser than 1 or bigger than 254
                for ( i = 0; i < ipparts.length; i++ ) {
                  if ( isNaN( ipparts[ i ] ) ) {
                    isIllegal = true
                    break
                  }
                  else  {
                    string2int = parseInt( ipparts[ i ] )

                    if ( string2int < 1 || string2int > 254 ) {
                      isIllegal = true
                        break
                    }
                  }                
                }  
                  
                if ( isIllegal ) {
                    messages.push( ERROR_IP_VALUE )                  
                } 
              } 
              // throw wrong IP address format error
              else {
                messages.push( ERROR_IP_FORMAT )
              }         
              
              // display error messages & stop form action
              if ( messages.length > 0 ) {
                // $( "#messageBoard" )[ 0 ].value = array2string( messages )
                $( "#messageBoard" )[ 0 ].innerText = array2string( messages )
                return false
              }     
            }
          ) 
        }  
      )  
    </script>
    <form id="editForm" name="editForm" action="pub" method="POST">
      [% SET row = edit.0 %]
      <input name="action" type="hidden" value="update" />
      <input name="pubid" type="hidden" value="[% row.pubid %]" />
      <h2 align="center" class="curTitle" >Edit Publisher #[% GET row.pubid %] <a href="javascript:window.history.back()"><img src="/uilib/comImg/back.png" border=0 width="25" height="25" /></a></h2>
       <div id="container">
    <div id="mainmenu">
					<ul id="tabs">
						<li>
							<a href="#EditPublisher">Edit Publisher</a>
						</li>
					</ul>
				<div>
				<div class="bar">&nbsp;</div>
				<div class="panel" id="EditPublisher">
      <div>  
        <table border="1">
          <thead>
            <tr>
              <th>Company</th>
              <th>URL</th>
              <th>Street</th>
              <th>City</th>
              <th>State</th>
              <th>ZIP</th>
              <th>Country</th>    
            </tr>
          </thead>          
          <tbody> 
            <tr>
              <td>
                <input id="company" name="company" class="Editable" type="text" value="[% GET row.company %]" />
              </td>
              <td>
                <input id="url" name="url" class="Editable" type="url" value="[% GET row.url %]" />
              </td>
              <td>
                <input id="street" name="street" class="Editable" type="text" value="[% GET row.street %]" />
              </td>
              <td>
                <input id="city" name="city" class="Editable" type="text" value="[% GET row.city %]" />
              </td>
              <td>
                <input id="state" name="state" class="Editable" type="text" value="[% GET row.state %]" />
              </td>
              <td>
                <input id="zip" name="zip" class="Editable" type="text" style="text-align: center" size="8" value="[% GET row.zip %]" />
              </td>
              <td>
                <input id="country" name="country" class="Editable" type="text" style="text-align: center" size="2" value="[% GET row.country %]" />
              </td>
            </tr>
          </tbody>  
        </table>
        <br/>        
        <table border="1">
          <thead>
            <tr>                            
              <th>Contact</th>
              <th>Contact&nbsp;Email</th>	
              <th>Phone (digits only)</th>
              <th>Fax (digits only)</th>
              <th>Time&nbsp;Zone</th>              
            </tr>
          </thead>              
          <tbody> 
            <tr>              
              <td>
                <input id="contact" name="contact" class="Editable" type="text" value="[% GET row.contact %]" />
              </td>
              <td>
                <input id="contactemail" name="contactemail" class="Editable" type="text" value="[% GET row.contactemail %]" />
              </td>
              <td>
                <input id="phone" name="phone" class="Editable" type="text" style="text-align: center" value="[% GET row.phone %]" />
              </td>
              <td>
                <input id="fax" name="fax" class="Editable" type="text" style="text-align: center" value="[% GET row.fax %]" />
              </td>
              <td style="text-align: center">
                <input id="timezone" name="timezone" class="Editable" type="text" style="text-align: center" size="2" value="[% GET row.timezone %]" />
              </td>              
            </tr>
          </tbody>  
        </table>              
        <br/>      
        <table border="1">
          <thead>
            <tr>
              <th>IP</th>
<!--              
              <th>Created</th>
-->              
              <th>Parent</th>
              <th>Service&nbsp;Level</th>
              <th>Visibility</th>
              <th>Access&nbsp;Order</th>
              <th>Status</th>                            
            </tr>
          </thead>              
          <tbody> 
            <tr>              
              <td>
                <input id="ip" name="ip" class="Editable" type="text" style="text-align: center" value="[% GET row.ip %]" />
              </td>
<!--              
              <td>
                <input id="created" name="created" class="Editable" type="text" style="text-align: center" value="[% GET row.created %]" />
              </td>
-->              
              <td>
                <input id="parent" name="parent" class="Editable" type="text" style="text-align: center" value="[% GET row.parent %]" />
              </td>
              <td>
                <select id="servicelevel" name="servicelevel" multiple="true">
                  <option value="Strategic" [% IF row.servicelevel == 'Strategic' %] selected="true" [% END %]>Strategic</option>
                  <option value="Standard" [% IF row.servicelevel == 'Standard' %] selected="true" [% END %]>Standard</option>
                  <option value="Self" [% IF row.servicelevel == 'Self' %] selected="true" [% END %]>Self</option>
                </select>
              </td>
              <td>
                <select id="visibility" name="visibility" multiple="true">
                  <option value="Blind" [% IF row.visibility == 'Blind' %] selected="true" [% END %]>Blind</option>
                  <option value="Standard" [% IF row.visibility == 'Standard' %] selected="true" [% END %]>Standard</option>
                  <option value="Open" [% IF row.visibility == 'Open' %] selected="true" [% END %]>Open</option>
                </select>
              </td>
              <td>
                <select id="accessorder" name="accessorder" multiple="true">
                  <option value="AllowDeny" [% IF row.accessorder == 'AllowDeny' %] selected="true" [% END %]>Allow, Deny</option>
                  <option value="DenyAllow" [% IF row.accessorder == 'DenyAllow' %] selected="true" [% END %]>Deny, Allow</option>
                  <option value="No" [% IF row.accessorder == 'No' %] selected="true" [% END %]>No</option>
                </select>
              </td>
              <td>
                <select id="status" name="status" multiple="true">
                  <option value="Yes" [% IF row.status == 'Yes' %] selected="true" [% END %]>Yes</option>
                  <option value="No" [% IF row.status == 'No' %] selected="true" [% END %]>No</option>
                  <option value="New" [% IF row.status == 'New' %] selected="true" [% END %]>New</option>
                </select>
              </td>              
            </tr>
          </tbody>  
        </table>
      </div>  
      <br/>
      <div align="center">
        <input type="reset" value="Reset" />
        &nbsp;
        <input id="btnSubmit" type="submit" value="Save" />
      </div>      
      <br/>
      <div id="messageBoard" style="align: center; text-align: left" />
      </div><!--<div class="panel" id="EditPublisher">-->
      </div><!--<div id="container">-->
    </form>
    [% INCLUDE form_ui_end.e %]
  </body>
</html>