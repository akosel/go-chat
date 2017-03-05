Vue.component("material-select", {
    template: '<select><slot></slot></select>',
    props: ['value'],
    watch: {
        value: function (value) {
            this.reload(value);
        }
    },
    methods:{
      reload : function (value) {
          var select = $(this.$el);

          select.val(value || this.value);
          select.material_select('destroy');
          select.material_select();
      }
    },
    mounted: function () {
        var vm = this;
        var select = $(this.$el);

        select
            .val(this.value)
            .on('change', function () {
                vm.$emit('input', this.value);
            });

        select.material_select();
    },
    updated: function () {
        this.reload();
    },
    destroyed: function () {
        $(this.$el).material_select('destroy');
    }
});

new Vue({
    el: '#app',

    data: {
        ws: null,
        newMsg: '',
        chatContent: '',
        email: null,
        username: null,
        roomName: '',
        joined: false,
	chatrooms: [{ text: 'main', value: 'main' }],
	users: [{ text: 'please login to chat', value: null }],
	activeRoom: 'main'
    },

    created: function() {
        var self = this;
        this.ws = new WebSocket('ws://' + window.location.host + '/ws');
        this.ws.addEventListener('message', function(e) {
            var msg = JSON.parse(e.data);
console.log(msg);
            if (msg.type === 'message') {
                self.chatContent += '<div class="chip">'
                    + '<img src="' + self.gravatarURL(msg.email) + '">'
                    + msg.username
                + '</div>'
                + emojione.toImage(msg.message) +'<br/>';

                var element = document.getElementById('chatroom-messages');
                element.scrollTop = element.scrollHeight;

                self.users = msg.users.map(function(user) {
                    return { text: user, value: user };
                });

		self.chatrooms = msg.rooms.map(function(room) {
		    return { text: room, value: room };
		});
            }
        });
    },

    methods: {
        send: function() {
            if (this.newMsg !== '') {
                this.ws.send(
                    JSON.stringify({
                        type: 'message',
			room: this.activeRoom,
                        email: this.email,
                        username: this.username,
                        message: $('<p>').html(this.newMsg).text()
                    })
                );
                this.newMsg = '';
            }
        },
        createUser: function() {
            if (!this.email) {
                Materialize.toast('You must enter an email', 2000);
                return;
            }
            if (!this.username) {
                Materialize.toast('You must choose a username', 2000);
                return;
            }
            this.email = $('<p>').html(this.email).text();
            this.username = $('<p>').html(this.username).text();
            this.joined = true;
	    this.ws.send(
		JSON.stringify({
		    type: 'createUser',
		    room: this.activeRoom,
		    email: this.email,
		    username: this.username
		})
	    )
	    this.changeRoom();
        },
        createRoom: function() {
            this.activeRoom = this.roomName;
	    this.chatrooms.push({ text: this.roomName, value: this.roomName });
            this.ws.send(
                JSON.stringify({
                    type: 'createRoom',
	            room: this.activeRoom
                })
            );
	    this.changeRoom();
	    this.roomName = '';
        },
	changeRoom: function() {
	    if (this.email && this.username) {
                this.ws.send(
                    JSON.stringify({
                        type: 'join',
	                room: this.activeRoom,
                        email: this.email,
                        username: this.username,
                        message: $('<p>').html(this.newMsg).text()
                    })
                );
	    }
	},
        gravatarURL: function(email) {
            return 'http://www.gravatar.com/avatar/' + CryptoJS.MD5(email);
        }
    }
});

