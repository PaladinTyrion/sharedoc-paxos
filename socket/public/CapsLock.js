/*

CapsLock.js

An object allowing the status of the caps lock key to be determined

Created by Stephen Morley - http://code.stephenmorley.org/ - and released under
the terms of the CC0 1.0 Universal legal code:

http://creativecommons.org/publicdomain/zero/1.0/legalcode

*/

// create the CapsLock object
var CapsLock = (function(){

  // initialise the status of the caps lock key
  var capsLock = false;

  // initialise the list of listeners
  var listeners = [];

  // store whether we are running on a Mac
  var isMac = /Mac/.test(navigator.platform);

  // Returns whether caps lock currently appears to be on.
  function isOn(){
    return capsLock;
  }

  /* Adds a listener. When a change is detected in the status of the caps lock
   * key the listener will be called, with a parameter of true if caps lock is
   * now on and false if caps lock is now off. The parameter is:
   *
   * listener - the listener
   */
  function addListener(listener){

    // add the listener to the list
    listeners.push(listener);

  }

  /* Handles a key press event. The parameter is:
   *
   * e - the event
   */
  function handleKeyPress(e){

    // ensure the event object is defined
    if (!e) e = window.event;

    // store the prior status of the caps lock key
    var priorCapsLock = capsLock;

    // determine the character code
    var charCode = (e.charCode ? e.charCode : e.keyCode);

    // store whether the caps lock key is down
    if (charCode >= 97 && charCode <= 122){
      capsLock = e.shiftKey;
    }else if (charCode >= 65 && charCode <= 90 && !(e.shiftKey && isMac)){
      capsLock = !e.shiftKey;
    }

    // call the listeners if the caps lock key status has changed
    if (capsLock != priorCapsLock){
      for (var index = 0; index < listeners.length; index ++){
        listeners[index](capsLock);
      }
    }

  }

  // listen for key press events
  if (window.addEventListener){
    window.addEventListener('keypress', handleKeyPress, false);
  }else{
    document.documentElement.attachEvent('onkeypress', handleKeyPress);
  }

  // return the public API
  return {
    isOn        : isOn,
    addListener : addListener
  };

})();
