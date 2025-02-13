import networkx as nx
import matplotlib.pyplot as plt

def plot_graph(dataframe):
    # Create a directed graph
    G = nx.DiGraph()

    #Root node
    G.add_node(-1, status="ROOT", label=dataframe.iloc[0]["seed_language"])

    # Add nodes and edges
    for index, entry in dataframe.iterrows():
        node_id = entry["edge_id"]
        parent_id = entry["parent_edge_id"]
        status = entry["status"]
        language = entry["target_languages"]

        # Add node with status attribute
        G.add_node(node_id, status=status, label=f"{language}")

        # Add edge if there is a valid parent
        G.add_edge(parent_id, node_id)

    # Define color map based on status
    color_map = {
        "TRANSLATION_FOUND": "green",
        "SKIPPED_TRANSLATION_FOUND": "grey",
        "SUCCESS": "blue",
        "TRANSLATED": "blue",
        "FAILED": "red",
        "SKIPPED_PARENT_FAILED": "grey",
        "SKIPPED_NO_EXTRACT": "grey",
        "FAILED_NO_EXTRACTED" : "red",
        "ROOT" : "skyblue"
    }

    # Get colors for nodes
    node_colors = [color_map[G.nodes[node]['status']] for node in G.nodes]

    # Draw the graph
    if nx.is_empty(G):
        print("The graph is empty.")
    else:
        # Draw the graph using graphviz layout for tree structure
        #pos = nx.nx_agraph.graphviz_layout(G, prog="dot")
        pos = nx.nx_agraph.graphviz_layout(G, prog="neato")

        # Draw nodes and edges with customized styles
        nx.draw_networkx_nodes(G, pos, node_color=node_colors, node_size=800, alpha=1)
        nx.draw_networkx_edges(G, pos, edge_color='#aaaaaa', arrows=True, width=1.5)
        nx.draw_networkx_labels(G, pos, labels=nx.get_node_attributes(G, 'label'), font_size=10, font_color='black')

        # Add edge labels
        edge_labels = nx.get_edge_attributes(G, 'label')
        nx.draw_networkx_edge_labels(G, pos, edge_labels=edge_labels, font_size=10)

        # Set plot style and show
        plt.rcParams['axes.facecolor'] = 'white'  # set background color
        plt.axis('off')  # turn off axis
        plt.title(f'ExpandTrans Executed Translation Tree ({dataframe.iloc[0]["seed_language"]} -> {dataframe.iloc[0]["request_target_language"]})', fontsize=14, fontweight='bold', pad=20)
        plt.tight_layout()
        plt.show()