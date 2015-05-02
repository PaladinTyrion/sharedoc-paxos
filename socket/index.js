var express = require("express");
var app = express();
var server = require("http").Server(app);
var io = require("socket.io")(server);

server.listen(8080, function(){
	console.log("listening on 8080");
});

//routing
app.use(express.static(__dirname + '/public'));

var seq = [];
var agreed_op = 0;
var modified_seq = [];

//event listener
io.on("connection",function(socket){
	//show connection successfully
	display(socket);

	//initialize newly connected client
	if (agreed_op > 0) {
		socket.emit('init_comt_op',modified_seq);
	};

	socket.on('op',function(op){
		seq.push(op);
		console.log(seq.length);
	});

	var interval = setInterval(function(){
		//check whether agreed op with seq's length
		if (agreed_op < seq.length) {
			//seq[agreed_op] -> modified_seq[agreed_op]
			var added = false;
			var current_op = seq[agreed_op];
			//inspect
			console.log(current_op);
			var version = current_op.Version;
			if (version == agreed_op) {
				modified_seq[agreed_op] = seq[agreed_op];
				added = true;
			}else if (version < agreed_op) {
				//update the version number
				for (var i = version; i < agreed_op; i++) {
					var sibling_op = seq[i];
					if (sibling_op.Type == "Insert" && current_op.Type == "Insert") {
						if (sibling_op.Position <= current_op.Position) {
							current_op.Position = current_op.Position + 1;
						};
					}else if (sibling_op.Type == "Delete" && current_op.Type == "Insert") {
						if (sibling_op.Position < current_op.Position) {
							current_op.Position = current_op.Position - 1;
						};
					}else if (sibling_op.Type == "Insert" && current_op.Type == "Delete") {
						if (sibling_op.Position <= current_op.Position) {
							current_op.Position = current_op.Position + 1;
						};
					}else if (sibling_op.Type == "Delete" && current_op.Type == "Delete") {
						if (sibling_op.Position < current_op.Position) {
							current_op.Position = current_op.Position - 1;
						}else if (sibling_op.Position == current_op.Position) {
							//discard this op, because it is already applied
							seq.splice(agreed_op,1);
							break;
						};
					};
					current_op.Version = current_op.Version + 1;
				};

				if (current_op.Version == agreed_op) {
					modified_seq[agreed_op] = current_op;
					added = true;
				};
			};


			if (added == true) {
				agreed_op = agreed_op+1;
				io.emit('op',current_op);
				//socket.broadcast.emit('op',current_op);
			};
			
			//inspect modified seq
			console.log(current_op);
		};

	},1);

	socket.on("disconnect",function(){
		console.log("user disconnected");
		clearInterval(interval);
	});
}); 

var display = function(socket){
	console.log("connection made");
};

