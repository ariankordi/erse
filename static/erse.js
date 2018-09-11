var PlaceList = {
	template: '#places-template',
	beforeMount: function() {
		this.$http.get('/api/places.jsp').then(function(response) {
			this.places = response.body;
			Vue.set(this, 'loading', false);
		});
	},
	data: function() {
		return {
			places: [],
			loading: true
		}
	}
}
var latestPost = {name: "", message: ""};
var connection;
var PostList = {
	template: '#posts-template',
	beforeMount: function() {
		this.$http.get('/api/posts.jsp?id=' + this.$route.params.id).then(function(response) {
			this.posts = response.body;
			Vue.set(this, 'loading', false);
			var posts = this.posts;
			connection = new WebSocket('ws' + ((location.protocol == 'https:') ? 's' : '') + '://' + location.host + '/api/postService.jsp?id=' + this.$route.params.id);
			connection.onmessage = function(event) {
				var message = JSON.parse(event.data);
				message.live = true;
				if(latestPost.name != message.name && latestPost.message != message.message) {
					posts.messages.unshift(message);
					Vue.set(posts, 'messages', posts.messages);
				}
			}
		});
	},
	beforeDestroy: function() {
		if(connection) {
    	connection.close();
		}
	},
	data: function() {
		return {
			posts: [],
			loading: true,
			noMessage: false,
			noName: false,
			sending: false,
			name: "",
			text: "",
			error: "",
			connection: null
		}
	},
	methods: {
		sendForm: function() {
			if(this.name == "") {
				Vue.set(this, 'noName', true);
				return;
			} else {
				Vue.set(this, 'noName', false);
			}
			if(this.text == "") {
				Vue.set(this, 'noMessage', true);
				return;
			} else {
				Vue.set(this, 'noMessage', false);
			}
			// CAN SEND!!!!!!!!!!!!!!!
			Vue.set(this, 'sending', true);
			Vue.set(this, 'error', "");
			latestPost = {name: this.name, message: this.text};

			this.$http.post('/api/createPost.jsp?id=' + this.$route.params.id, {
				name: this.name,
				message: this.text
			}, {
				emulateJSON: true
			}).catch(function(response) {
				Vue.set(this, 'sending', false);
				Vue.set(this, 'error', "ERROR! " + response.status + " " + response.statusText);
			}).then(function(response) {
				if(!this.error) {
					Vue.set(this, 'sending', false);
					// unshift is like push but at the end
					this.posts.messages.unshift({
						id: response.data.id,
						name: this.name,
						message: this.text,
						likes: 0
					});
					Vue.set(this, 'text', "");
				}
			});
		}
	}
}

var router = new VueRouter({
	mode: 'history',
  routes: [
		{ path: '/', component: PlaceList },
		{ path: '/place/:id', component: PostList }
	]
});

var app = new Vue({
  router: router
}).$mount('#app');
